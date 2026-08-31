#ifndef MQTT_CLIENT_H
#define MQTT_CLIENT_H

#include <WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include "config.h"
#include "sensor_stubs.h"

// SystemMQTTClient manages MQTT broker connection, telemetry publishing, and health-check reports
class SystemMQTTClient {
public:
    SystemMQTTClient();
    
    // Initializes topic routes and sets broker address
    void init();
    
    // Connects to MQTT broker using unique MAC address as client ID
    bool connect();
    
    // Keeps MQTT client loop active and manages auto-reconnection
    void loop();
    
    // Serializes and publishes BMP280 sensor readings payload to telemetry topic
    bool publishTelemetry(const SensorPayload& data);
    
    // Serializes and publishes device operational status to health-check topic
    bool publishHealthCheck(bool sensorOk);
    
    String getClientId() const { return clientId; }
    String getTelemetryTopic() const { return telemetryTopic; }
    String getHealthCheckTopic() const { return healthCheckTopic; }

private:
    WiFiClient espClient;
    PubSubClient mqttClient;
    
    String clientId;
    String telemetryTopic;
    String healthCheckTopic;
    String commandTopic;
    String broadcastTopic;

    // Static callback for incoming MQTT messages
    static void mqttCallback(char* topic, byte* payload, unsigned int length);
};

extern SystemMQTTClient mqttService;

#endif // MQTT_CLIENT_H
