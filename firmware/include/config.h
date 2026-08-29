#ifndef CONFIG_H
#define CONFIG_H

#include <Arduino.h>

// WiFi Default Credentials (Overridden if build flags from secrets.ini are defined)
#ifndef DEFAULT_WIFI_SSID
#define DEFAULT_WIFI_SSID       "YourWiFiSSID"
#endif

#ifndef DEFAULT_WIFI_PASS
#define DEFAULT_WIFI_PASS       "YourWiFiPass"
#endif

// MQTT Broker Credentials (Overridden if build flags from secrets.ini are defined)
#ifndef MQTT_BROKER_HOST
#define MQTT_BROKER_HOST        "192.168.1.100"
#endif

#ifndef MQTT_BROKER_PORT
#define MQTT_BROKER_PORT        1883
#endif

#ifndef MQTT_USER
#define MQTT_USER               ""
#endif

#ifndef MQTT_PASS
#define MQTT_PASS               ""
#endif

// Centralized MQTT Topic Definitions (Overridden if build flags from secrets.ini are defined)
#ifndef MQTT_TOPIC_BASE
#define MQTT_TOPIC_BASE         "devices"
#endif

#define MQTT_TOPIC_TELEMETRY    "telemetry"
#define MQTT_TOPIC_HEALTHCHECK  "health-check"
#define MQTT_TOPIC_COMMANDS     "commands"
#define MQTT_TOPIC_BROADCAST    "broadcast"

// Network & Captive Portal Timeouts
#define WIFI_CONNECT_TIMEOUT    15000
#define CAPTIVE_PORTAL_AP_NAME  "ESP32-Setup-Portal"

// Periodic Task Execution Intervals (Overridden if build flags from secrets.ini are defined)
#ifndef SENSOR_READ_INTERVAL_MS
#define SENSOR_READ_INTERVAL_MS  5000
#endif

#ifndef HEALTH_CHECK_INTERVAL_MS
#define HEALTH_CHECK_INTERVAL_MS 30000
#endif

// ESP32 Hardware Pin Mapping & FreeRTOS Resource Definitions
#define I2C_SDA                 21
#define I2C_SCL                 22
#define SENSOR_TASK_STACK_SIZE  4096
#define NETWORK_TASK_STACK_SIZE 8192
#define SENSOR_QUEUE_LEN        10

// Returns unique hardware MAC address string used as node identifier
inline String getDeviceMacId() {
    uint64_t mac = ESP.getEfuseMac();
    char macStr[13];
    snprintf(macStr, sizeof(macStr), "%04X%08X", (uint16_t)(mac >> 32), (uint32_t)mac);
    return String(macStr);
}

#endif // CONFIG_H
