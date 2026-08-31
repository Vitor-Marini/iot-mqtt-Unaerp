#include <Arduino.h>
#include "config.h"
#include "wifi_manager.h"
#include "mqtt_client.h"
#include "sensor_stubs.h"

QueueHandle_t xSensorQueue = NULL;

TaskHandle_t hTaskSensors  = NULL;
TaskHandle_t hTaskWiFiMQTT = NULL;

void vTaskSensors(void* pvParameters) {
    TickType_t xLastWakeTime = xTaskGetTickCount();
    const TickType_t xFrequency = pdMS_TO_TICKS(SENSOR_READ_INTERVAL_MS);

    for (;;) {
        vTaskDelayUntil(&xLastWakeTime, xFrequency);

        SensorPayload payload = sensorService.readData();

        if (xSensorQueue != NULL) {
            if (xQueueSend(xSensorQueue, &payload, 0) != pdPASS) {
                Serial.println("[WARNING] Sensor queue full. Payload dropped.");
            }
        }
    }
}

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
            Serial.println("[WIFI] Waiting for network reconnection...");
            vTaskDelay(pdMS_TO_TICKS(2000));
        }

        vTaskDelay(pdMS_TO_TICKS(10));
    }
}

void setup() {
    Serial.begin(115200);
    delay(1000);

    Serial.println("\n==================================================");
    Serial.println("       ESP32 BMP280 MQTT NODE FIRMWARE           ");
    Serial.println("==================================================");
    Serial.println(" [SYSTEM] MAC Address  : " + getDeviceMacId());
    Serial.println(" [SYSTEM] Topic Base   : " + String(MQTT_TOPIC_BASE));
    Serial.println(" [SYSTEM] Broker Host  : " + String(MQTT_BROKER_HOST) + ":" + String(MQTT_BROKER_PORT));
    Serial.println(" [SYSTEM] I2C Pins     : SDA=" + String(I2C_SDA) + ", SCL=" + String(I2C_SCL));
    Serial.println(" [SYSTEM] Read Interval: " + String(SENSOR_READ_INTERVAL_MS) + " ms");
    Serial.println(" [SYSTEM] Health Interval: " + String(HEALTH_CHECK_INTERVAL_MS) + " ms");
    Serial.println("==================================================\n");

    // Initialize hardware sensor synchronously before FreeRTOS tasks start
    sensorService.init();

    xSensorQueue = xQueueCreate(SENSOR_QUEUE_LEN, sizeof(SensorPayload));
    if (xSensorQueue == NULL) {
        Serial.println("[ERROR] Failed to create FreeRTOS sensor queue!");
        return;
    }

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

    Serial.println("[SYSTEM] FreeRTOS tasks created successfully.\n");
}

void loop() {
    vTaskDelete(NULL);
}
