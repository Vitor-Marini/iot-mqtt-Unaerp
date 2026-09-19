#include <Arduino.h>
#include "config.h"
#include "wifi_manager.h"
#include "mqtt_client.h"
#include "sensor_stubs.h"
#include "ota_manager.h"

// Filas FreeRTOS para comunicacao inter-tarefas
QueueHandle_t xSensorQueue = NULL;
QueueHandle_t xOTAQueue    = NULL;

// Handles das tarefas FreeRTOS
TaskHandle_t hTaskSensors  = NULL;
TaskHandle_t hTaskWiFiMQTT = NULL;
TaskHandle_t hTaskOTA      = NULL;

// ==============================================================================
// 🔴 TAREFA 1: Leitura Periódica dos Sensores (Core 1)
// ==============================================================================
void vTaskSensors(void* pvParameters) {
    TickType_t xLastWakeTime = xTaskGetTickCount();
    const TickType_t xFrequency = pdMS_TO_TICKS(SENSOR_READ_INTERVAL_MS);

    for (;;) {
        vTaskDelayUntil(&xLastWakeTime, xFrequency);

        // Se o OTA estiver em andamento, pausa leituras para preservar recursos de CPU e RAM
        if (otaService.isUpdating()) {
            continue;
        }

        SensorPayload payload = sensorService.readData();

        if (xSensorQueue != NULL) {
            if (xQueueSend(xSensorQueue, &payload, 0) != pdPASS) {
                Serial.println("[WARNING] Sensor queue full. Payload dropped.");
            }
        }
    }
}

// ==============================================================================
// 🟢 TAREFA 2: Gestão de Rede WiFi e Cliente MQTT (Core 1)
// ==============================================================================
void vTaskWiFiMQTT(void* pvParameters) {
    if (!wifiManagerService.initWiFi()) {
        Serial.println("[ERROR] Critical WiFi initialization failure!");
    }

    mqttService.init();

    SensorPayload receivedPayload;
    static uint32_t lastHealthCheck = 0;

    for (;;) {
        wifiManagerService.processWiFi();

        if (wifiManagerService.isConnected()) {
            // Só executa o loop MQTT e telemetria se NÃO estiver no meio de uma atualização OTA
            if (!otaService.isUpdating()) {
                mqttService.loop();

                if (xQueueReceive(xSensorQueue, &receivedPayload, pdMS_TO_TICKS(50)) == pdPASS) {
                    mqttService.publishTelemetry(receivedPayload);
                }

                uint32_t now = millis();
                if (now - lastHealthCheck >= HEALTH_CHECK_INTERVAL_MS) {
                    lastHealthCheck = now;
                    mqttService.publishHealthCheck(receivedPayload.isValid);
                }
            } else {
                // Em modo de atualizacao OTA, desacelera o loop WiFi para dedicar a CPU e rede ao download
                vTaskDelay(pdMS_TO_TICKS(500));
            }
        } else {
            Serial.println("[WIFI] Waiting for network reconnection...");
            vTaskDelay(pdMS_TO_TICKS(2000));
        }

        vTaskDelay(pdMS_TO_TICKS(10));
    }
}

// ==============================================================================
// 🔵 TAREFA 3: Serviço OTA em Segundo Plano (Core 0)
// ==============================================================================
void vTaskOTA(void* pvParameters) {
    Serial.println("[FreeRTOS] vTaskOTA iniciada no Core 0.");

    // Aguarda o WiFi conectar antes de subir o servidor ArduinoOTA local
    while (!wifiManagerService.isConnected()) {
        vTaskDelay(pdMS_TO_TICKS(1000));
    }

    otaService.init();

    OTACommandRequest req;

    for (;;) {
        // Trata requisicoes do ArduinoOTA local
        otaService.handle();

        // Cede tempo para IDLE0 rodar quando ocioso
        vTaskDelay(pdMS_TO_TICKS(10));

        // Verifica se ha comando de HTTP OTA disparado via MQTT
        if (xQueueReceive(xOTAQueue, &req, pdMS_TO_TICKS(100)) == pdPASS) {
            Serial.println("\n[vTaskOTA] Recebido comando de atualizacao OTA!");
            Serial.printf("[vTaskOTA] Baixando versao %s de: %s\n", req.version, req.url);

            // Jitter aleatório pequeno (0 a 1.5s) baseado no MAC
            // Evita que todas as placas batam o SYN de abertura de socket TCP exatamente no mesmo milissegundo no AP Wi-Fi
            uint32_t jitterMs = (uint32_t)(ESP.getEfuseMac() % 1500);
            vTaskDelay(pdMS_TO_TICKS(jitterMs));

            otaService.performHTTPUpdate(String(req.url), String(req.version), String(req.md5));
        }
    }
}

// ==============================================================================
// 🚀 SETUP & INITIALIZATION
// ==============================================================================
void setup() {
    Serial.begin(115200);
    delay(1000);

    Serial.println("\n==================================================");
    Serial.println("       ESP32 BMP280 MQTT NODE FIRMWARE           ");
    Serial.println("==================================================");
    Serial.println(" [SYSTEM] Firmware Ver : " + String(FIRMWARE_VERSION));
    Serial.println(" [SYSTEM] MAC Address  : " + getDeviceMacId());
    Serial.println(" [SYSTEM] Topic Base   : " + String(MQTT_TOPIC_BASE));
    Serial.println(" [SYSTEM] Broker Host  : " + String(MQTT_BROKER_HOST) + ":" + String(MQTT_BROKER_PORT));
    Serial.println(" [SYSTEM] I2C Pins     : SDA=" + String(I2C_SDA) + ", SCL=" + String(I2C_SCL));
    Serial.println(" [SYSTEM] Read Interval: " + String(SENSOR_READ_INTERVAL_MS) + " ms");
    Serial.println(" [SYSTEM] Health Intvl : " + String(HEALTH_CHECK_INTERVAL_MS) + " ms");
    Serial.println("==================================================\n");

    // Inicializa hardware do sensor antes das tarefas FreeRTOS
    sensorService.init();

    // 1. Criar Filas FreeRTOS
    xSensorQueue = xQueueCreate(SENSOR_QUEUE_LEN, sizeof(SensorPayload));
    if (xSensorQueue == NULL) {
        Serial.println("[ERROR] Failed to create FreeRTOS sensor queue!");
        return;
    }

    xOTAQueue = xQueueCreate(2, sizeof(OTACommandRequest));
    if (xOTAQueue == NULL) {
        Serial.println("[ERROR] Failed to create FreeRTOS OTA queue!");
        return;
    }

    // 2. Criar Tarefas FreeRTOS nos Cores Adequados
    xTaskCreatePinnedToCore(
        vTaskSensors,
        "SensorsTask",
        SENSOR_TASK_STACK_SIZE,
        NULL,
        1,
        &hTaskSensors,
        1
    );

    xTaskCreatePinnedToCore(
        vTaskWiFiMQTT,
        "WiFiMQTTTask",
        NETWORK_TASK_STACK_SIZE,
        NULL,
        2,
        &hTaskWiFiMQTT,
        1
    );

    xTaskCreatePinnedToCore(
        vTaskOTA,
        "OTATask",
        OTA_TASK_STACK_SIZE,
        NULL,
        1,
        &hTaskOTA,
        1 // Core 1 para não competir com a stack de Wi-Fi e evitar starvation do IDLE0
    );

    Serial.println("[SYSTEM] FreeRTOS tasks (Sensors, WiFi/MQTT, OTA) created successfully.\n");
}

void loop() {
    vTaskDelete(NULL);
}
