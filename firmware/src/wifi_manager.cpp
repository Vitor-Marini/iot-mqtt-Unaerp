#include "wifi_manager.h"

SystemWiFiManager wifiManagerService;

SystemWiFiManager::SystemWiFiManager() : connected(false) {}

bool SystemWiFiManager::initWiFi() {
    WiFi.mode(WIFI_STA);
    Serial.println("[WiFi] Initializing WiFi subsystem...");

    wm.setConnectTimeout(WIFI_CONNECT_TIMEOUT / 1000);
    wm.setConfigPortalTimeout(180);
    wm.setSaveConnectTimeout(15);
    
    String apName = String(CAPTIVE_PORTAL_AP_NAME) + "-" + getDeviceMacId().substring(6);

    Serial.print("[WiFi] Connecting to known network or AP Portal: ");
    Serial.println(apName);

    // Creates an OPEN Access Point (no password required for portal access)
    if (!wm.autoConnect(apName.c_str())) {
        Serial.println("[WiFi Error] Connection failed and Captive Portal timed out.");
        connected = false;
        return false;
    }

    Serial.print("[WiFi] Connected successfully. IP Address: ");
    Serial.println(WiFi.localIP());
    
    configTime(0, 0, "pool.ntp.org", "time.nist.gov");
    Serial.println("[NTP] UTC time synchronization initiated.");
    
    connected = true;
    return true;
}

void SystemWiFiManager::processWiFi() {
    wm.process();
    if (WiFi.status() != WL_CONNECTED) {
        connected = false;
    } else {
        connected = true;
    }
}

bool SystemWiFiManager::isConnected() {
    return (WiFi.status() == WL_CONNECTED);
}

String SystemWiFiManager::getIP() {
    return WiFi.localIP().toString();
}
