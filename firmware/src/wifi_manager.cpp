#include "wifi_manager.h"

SystemWiFiManager wifiManagerService;

SystemWiFiManager::SystemWiFiManager() : connected(false) {}

bool SystemWiFiManager::initWiFi() {
    WiFi.mode(WIFI_STA);
    Serial.println("[WiFi] Initializing WiFi subsystem...");

    String ssid = DEFAULT_WIFI_SSID;
    String pass = DEFAULT_WIFI_PASS;

    // 1. Attempt connection using pre-configured build credentials (from secrets.ini) if defined
    if (ssid.length() > 0 && ssid != "YourWiFiSSID") {
        Serial.println("[WiFi] Attempting direct connection to configured SSID: " + ssid);
        WiFi.begin(ssid.c_str(), pass.c_str());
        
        uint32_t startAttempt = millis();
        while (WiFi.status() != WL_CONNECTED && millis() - startAttempt < 15000) {
            delay(500);
            Serial.print(".");
        }
        Serial.println();

        if (WiFi.status() == WL_CONNECTED) {
            Serial.print("[WiFi] Direct connection SUCCESSFUL. IP Address: ");
            Serial.println(WiFi.localIP());
            
            configTime(0, 0, "pool.ntp.org", "time.nist.gov");
            Serial.println("[NTP] UTC time synchronization initiated.");
            
            connected = true;
            return true;
        }
        Serial.println("[WiFi Warning] Direct connection timed out. Falling back to WiFiManager / Captive Portal...");
    }

    // 2. Fallback to WiFiManager autoConnect (stored NVS network or open Captive Portal)
    wm.setConnectTimeout(30);
    wm.setConfigPortalTimeout(180);
    wm.setSaveConnectTimeout(30);
    
    String apName = String(CAPTIVE_PORTAL_AP_NAME) + "-" + getDeviceMacId().substring(6);

    Serial.print("[WiFi] Starting WiFiManager autoConnect / AP: ");
    Serial.println(apName);

    if (!wm.autoConnect(apName.c_str())) {
        Serial.println("[WiFi Error] Connection failed and Captive Portal timed out.");
        connected = false;
        return false;
    }

    Serial.print("[WiFi] Connected via WiFiManager. IP Address: ");
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
