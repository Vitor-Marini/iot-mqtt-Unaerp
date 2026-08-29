# OTA (Over-The-Air) Firmware Update Implementation Guide

This document outlines the design, architecture, and step-by-step implementation guide for adding HTTP-based OTA updates to the ESP32 firmware via MQTT.

---

## 1. Overview & Architecture

The OTA system allows remote firmware updates without physical access to the ESP32 board.

### Workflow:
1. **Compilation**: Compile the firmware binary (`firmware.bin`) in PlatformIO.
2. **File Server**: Host `firmware.bin` on a local or remote HTTP server (e.g., Python HTTP Server, Nginx, or AWS S3).
3. **MQTT Trigger**: Send a JSON command payload to the node's MQTT command topic.
4. **Download & Flash**: ESP32 downloads the binary chunk-by-chunk using `HTTPUpdate` and flashes it into the secondary OTA partition.
5. **Reboot & Verify**: Board reboots into the new partition. Upon successful MQTT connection, the app validates the partition and cancels automatic rollback.

---

## 2. MQTT Command Specification

### Topic Format:
`<base_topic>/{MAC}/commands` (e.g., `devices/{MAC}/commands`)

### Payload Format:
```json
{
  "ota_url": "http://192.168.1.100:8000/firmware.bin"
}
```

---

## 3. Local HTTP Server Setup

To serve firmware updates locally during testing:

### Option A: Python Built-in HTTP Server
Run inside the directory containing `firmware.bin`:
```bash
python3 -m http.server 8000
```
Your URL will be `http://<YOUR_COMPUTER_IP>:8000/firmware.bin`.

---

## 4. ESP32 Code Implementation Reference

If you decide to re-enable OTA in the future, implement the following components:

### A. Include Required Headers
```cpp
#include <HTTPClient.h>
#include <HTTPUpdate.h>
#include <esp_ota_ops.h>
```

### B. HTTP Update Function
```cpp
bool performHTTPUpdate(const String& url) {
    if (url.length() == 0) return false;

    WiFiClient client;
    t_httpUpdate_return ret = httpUpdate.update(client, url);

    switch (ret) {
        case HTTP_UPDATE_FAILED:
            Serial.printf("HTTP OTA Failed (%d): %s\n", 
                          httpUpdate.getLastError(), 
                          httpUpdate.getLastErrorString().c_str());
            return false;

        case HTTP_UPDATE_NO_UPDATES:
            Serial.println("No update available.");
            return false;

        case HTTP_UPDATE_OK:
            Serial.println("Firmware update successful. Rebooting...");
            return true;
    }
    return false;
}
```

### C. Partition Verification & Rollback Prevention
Call this function inside your `SystemMQTTClient::connect()` method once MQTT is connected successfully:

```cpp
void confirmFirmwareValidity() {
    const esp_partition_t* running = esp_ota_get_running_partition();
    esp_ota_img_states_t ota_state;

    if (esp_ota_get_state_partition(running, &ota_state) == ESP_OK) {
        if (ota_state == ESP_OTA_IMG_PENDING_VERIFY) {
            Serial.println("New firmware verified. Automatic rollback cancelled.");
            esp_ota_mark_app_valid_cancel_rollback();
        }
    }
}
```

### D. Integration in MQTT Callback
```cpp
void mqttCallback(char* topic, byte* payload, unsigned int length) {
    String message;
    for (unsigned int i = 0; i < length; i++) {
        message += (char)payload[i];
    }
    
    JsonDocument doc;
    DeserializationError error = deserializeJson(doc, message);
    if (!error && doc["ota_url"].is<const char*>()) {
        String otaUrl = doc["ota_url"].as<String>();
        performHTTPUpdate(otaUrl);
    }
}
```
