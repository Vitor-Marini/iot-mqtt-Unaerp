#include "mqtt_client.h"
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

    mqttClient.setServer(MQTT_BROKER_HOST, MQTT_BROKER_PORT);
    mqttClient.setCallback(mqttCallback);

    Serial.println("[MQTT] Client Routes Configured:");
    Serial.println("       Device ID (MAC)   : " + clientId);
    Serial.println("       Telemetry Topic   : " + telemetryTopic);
    Serial.println("       Health Check Topic: " + healthCheckTopic);
    Serial.println("       Command Topic     : " + commandTopic);
}

bool SystemMQTTClient::connect() {
    if (mqttClient.connected()) return true;

    Serial.print("[MQTT] Connecting to Broker (" + String(MQTT_BROKER_HOST) + ":" + String(MQTT_BROKER_PORT) + ")... ");
    
    if (mqttClient.connect(clientId.c_str(), MQTT_USER, MQTT_PASS)) {
        Serial.println("CONNECTED!");
        
        mqttClient.subscribe(commandTopic.c_str());
        mqttClient.subscribe(broadcastTopic.c_str());

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
    doc["status"]       = sensorOk ? "OK" : "ERROR";
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

void SystemMQTTClient::mqttCallback(char* topic, byte* payload, unsigned int length) {
    String message;
    for (unsigned int i = 0; i < length; i++) {
        message += (char)payload[i];
    }
    Serial.printf("[MQTT RX] Topic: %s | Message: %s\n", topic, message.c_str());
}
