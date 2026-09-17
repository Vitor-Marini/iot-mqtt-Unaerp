#include "mqtt_client.h"
#include "ota_manager.h"
#include <time.h>

SystemMQTTClient mqttService;

SystemMQTTClient::SystemMQTTClient() : mqttClient(espClient) {}

void SystemMQTTClient::init() {
    String macId = getDeviceMacId();
    clientId = macId;

    // Build dynamic topic paths using base path, MAC address, and configured topic suffixes
    telemetryTopic   = String(MQTT_TOPIC_BASE) + "/" + macId + "/" + String(MQTT_TOPIC_TELEMETRY);
    healthCheckTopic = String(MQTT_TOPIC_BASE) + "/" + macId + "/" + String(MQTT_TOPIC_HEALTHCHECK);
    commandTopic     = String(MQTT_TOPIC_BASE) + "/" + macId + "/" + String(MQTT_TOPIC_COMMANDS);
    broadcastTopic   = String(MQTT_TOPIC_BASE) + "/" + String(MQTT_TOPIC_BROADCAST);
    otaStatusTopic   = String(MQTT_TOPIC_BASE) + "/" + macId + "/" + String(MQTT_TOPIC_OTA_STATUS);

    mqttClient.setServer(MQTT_BROKER_HOST, MQTT_BROKER_PORT);
    mqttClient.setCallback(mqttCallback);

    Serial.println("[MQTT] Client Routes Configured:");
    Serial.println("       Device ID (MAC)   : " + clientId);
    Serial.println("       Telemetry Topic   : " + telemetryTopic);
    Serial.println("       Health Check Topic: " + healthCheckTopic);
    Serial.println("       Command Topic     : " + commandTopic);
    Serial.println("       OTA Status Topic  : " + otaStatusTopic);
}

bool SystemMQTTClient::connect() {
    if (mqttClient.connected()) return true;

    Serial.print("[MQTT] Connecting to Broker (" + String(MQTT_BROKER_HOST) + ":" + String(MQTT_BROKER_PORT) + ")... ");
    
    if (mqttClient.connect(clientId.c_str(), MQTT_USER, MQTT_PASS)) {
        Serial.println("CONNECTED!");
        
        mqttClient.subscribe(commandTopic.c_str());
        mqttClient.subscribe(broadcastTopic.c_str());

        // Confirma saude do novo firmware para cancelar rollback do bootloader
        otaService.confirmFirmwareValidity();

        publishHealthCheck(true);
        return true;
    } else {
        Serial.print("FAILED (rc=");
        Serial.print(mqttClient.state());
        Serial.println(")");
        return false;
    }
}

void SystemMQTTClient::loop() {
    if (!mqttClient.connected()) {
        static uint32_t lastReconnectAttempt = 0;
        uint32_t now = millis();
        if (now - lastReconnectAttempt > 5000) {
            lastReconnectAttempt = now;
            connect();
        }
    } else {
        mqttClient.loop();
    }
}

bool SystemMQTTClient::publishTelemetry(const SensorPayload& data) {
    if (!mqttClient.connected()) return false;

    time_t now = time(NULL);
    uint64_t timestampUtc = (now > 1000000000) ? (uint64_t)now : (uint64_t)(data.timestamp_ms / 1000);

    JsonDocument doc;
    doc["sensor_id"]    = getDeviceMacId();
    doc["sensor_model"] = "BMP280";
    doc["temperature"]  = data.temperature;
    doc["pressure"]     = data.pressure;
    doc["altitude"]     = data.altitude;
    doc["timestamp"]    = timestampUtc;

    char buffer[256];
    size_t n = serializeJson(doc, buffer);

    bool result = mqttClient.publish(telemetryTopic.c_str(), buffer, n);
    if (result) {
        Serial.println("\n--------------------------------------------------");
        Serial.println("[MQTT TX] Telemetry Published -> " + telemetryTopic);
        serializeJsonPretty(doc, Serial);
        Serial.println("\n--------------------------------------------------");
    } else {
        Serial.println("[MQTT ERROR] Failed to publish telemetry payload.");
    }
    return result;
}

bool SystemMQTTClient::publishHealthCheck(bool sensorOk) {
    if (!mqttClient.connected()) return false;

    time_t now = time(NULL);
    uint64_t timestampUtc = (now > 1000000000) ? (uint64_t)now : (uint64_t)(millis() / 1000);

    JsonDocument doc;
    doc["sensor_id"]    = getDeviceMacId();
    doc["sensor_model"] = "BMP280";
    doc["version"]      = FIRMWARE_VERSION;
    doc["status"]       = sensorOk ? "OK" : "ERROR";
    doc["ip"]           = WiFi.localIP().toString();
    doc["rssi"]         = WiFi.RSSI();
    doc["free_heap"]    = ESP.getFreeHeap();
    doc["uptime_ms"]    = millis();
    doc["timestamp"]    = timestampUtc;

    char buffer[256];
    size_t n = serializeJson(doc, buffer);

    bool result = mqttClient.publish(healthCheckTopic.c_str(), buffer, n);
    if (result) {
        Serial.println("\n--------------------------------------------------");
        Serial.println("[MQTT TX] Health-Check Published -> " + healthCheckTopic);
        serializeJsonPretty(doc, Serial);
        Serial.println("\n--------------------------------------------------");
    } else {
        Serial.println("[MQTT ERROR] Failed to publish health-check payload.");
    }
    return result;
}

bool SystemMQTTClient::publishOTAStatus(const String& status, const String& version, const String& message, const String& error) {
    if (!mqttClient.connected()) return false;

    time_t now = time(NULL);
    uint64_t timestampUtc = (now > 1000000000) ? (uint64_t)now : (uint64_t)(millis() / 1000);

    JsonDocument doc;
    doc["sensor_id"]  = getDeviceMacId();
    doc["status"]     = status;
    doc["version"]    = version;
    doc["ip"]         = WiFi.localIP().toString();
    doc["message"]    = message;
    if (error.length() > 0) {
        doc["error"]  = error;
    }
    doc["free_heap"]  = ESP.getFreeHeap();
    doc["uptime_ms"]  = millis();
    doc["timestamp"]  = timestampUtc;

    char buffer[384];
    serializeJson(doc, buffer);

    bool result = mqttClient.publish(otaStatusTopic.c_str(), buffer, false);
    if (result) {
        Serial.println("\n[MQTT TX] OTA Status Published -> " + otaStatusTopic + " [" + status + "]");
    }
    return result;
}

void SystemMQTTClient::mqttCallback(char* topic, byte* payload, unsigned int length) {
    String message;
    for (unsigned int i = 0; i < length; i++) {
        message += (char)payload[i];
    }
    Serial.printf("[MQTT RX] Topic: %s | Message: %s\n", topic, message.c_str());

    // Processa comandos JSON recebidos
    JsonDocument doc;
    DeserializationError err = deserializeJson(doc, message);
    if (!err) {
        String cmd = doc["cmd"].as<String>();
        String otaUrl = "";
        if (doc["url"].is<const char*>()) {
            otaUrl = doc["url"].as<String>();
        } else if (doc["ota_url"].is<const char*>()) {
            otaUrl = doc["ota_url"].as<String>();
        }

        // Se for comando OTA ou contiver URL de firmware
        if (cmd == "ota" || otaUrl.length() > 0) {
            String version = doc["version"].is<const char*>() ? doc["version"].as<String>() : "latest";
            String md5 = doc["md5"].is<const char*>() ? doc["md5"].as<String>() : "";

            Serial.println("[MQTT OTA] Comando de atualizacao recebido! Preparando requisicao...");

            if (xOTAQueue != NULL) {
                OTACommandRequest req;
                memset(&req, 0, sizeof(req));
                strncpy(req.url, otaUrl.c_str(), sizeof(req.url) - 1);
                strncpy(req.version, version.c_str(), sizeof(req.version) - 1);
                strncpy(req.md5, md5.c_str(), sizeof(req.md5) - 1);

                if (xQueueSend(xOTAQueue, &req, 0) == pdPASS) {
                    Serial.println("[MQTT OTA] Requisicao enviada com sucesso para vTaskOTA!");
                } else {
                    Serial.println("[MQTT OTA ERROR] Fila de OTA cheia! Comando descartado.");
                }
            } else {
                Serial.println("[MQTT OTA ERROR] Fila xOTAQueue nao inicializada!");
            }
        }
    }
}
