#include "wifi_manager.h"
#include <time.h>

SystemWiFiManager wifiManagerService;

SystemWiFiManager::SystemWiFiManager() : connected(false) {}

static void syncNTPTime() {
    Serial.print("[NTP] Syncing UTC network time");
    configTime(0, 0, "pool.ntp.org", "time.nist.gov");

    time_t now = time(NULL);
    uint32_t startWait = millis();
    while (now < 1000000000 && millis() - startWait < 5000) {
        delay(250);
        Serial.print(".");
        now = time(NULL);
    }
    Serial.println();

    if (now > 1000000000) {
        struct tm timeinfo;
        gmtime_r(&now, &timeinfo);
        char timeStr[64];
        strftime(timeStr, sizeof(timeStr), "%Y-%m-%d %H:%M:%S UTC", &timeinfo);
        Serial.printf("[NTP] Time synchronized: %s (Epoch: %ld)\n", timeStr, (long)now);
    } else {
        Serial.println("[NTP WARNING] Sync timeout. Will use fallback timestamps.");
    }
}

bool SystemWiFiManager::initWiFi() {
    WiFi.mode(WIFI_STA);
    Serial.println("[WIFI] Initializing WiFi subsystem...");

    String ssid = DEFAULT_WIFI_SSID;
    String pass = DEFAULT_WIFI_PASS;

    // 1. Direct connection attempt using secrets.ini credentials if defined
    if (ssid.length() > 0 && ssid != "YourWiFiSSID") {
        Serial.print("[WIFI] Connecting to '" + ssid + "' ");
        WiFi.begin(ssid.c_str(), pass.c_str());
        
        uint32_t startAttempt = millis();
        while (WiFi.status() != WL_CONNECTED && millis() - startAttempt < 15000) {
            delay(500);
            Serial.print(".");
        }
        Serial.println();

        if (WiFi.status() == WL_CONNECTED) {
            Serial.println("[WIFI] Connected! IP Address: " + WiFi.localIP().toString());
            syncNTPTime();
            connected = true;
            return true;
        }
        Serial.println("[WIFI] Direct connection timed out. Fallback to WiFiManager Portal...");
    }

    // 2. Fallback to WiFiManager autoConnect (stored NVS or open Captive Portal)
    wm.setConnectTimeout(30);
    wm.setConfigPortalTimeout(180);
    wm.setSaveConnectTimeout(30);
    
    String apName = String(CAPTIVE_PORTAL_AP_NAME) + "-" + getDeviceMacId().substring(6);
    Serial.println("[WIFI] Starting Captive Portal AP: " + apName);

    if (!wm.autoConnect(apName.c_str())) {
        Serial.println("[WIFI ERROR] Portal connection failed or timed out.");
        connected = false;
        return false;
    }

    Serial.println("[WIFI] Connected via Portal! IP Address: " + WiFi.localIP().toString());
    syncNTPTime();
    
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
