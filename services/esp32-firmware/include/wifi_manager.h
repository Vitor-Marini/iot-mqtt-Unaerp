#ifndef WIFI_MANAGER_H
#define WIFI_MANAGER_H

#include <WiFi.h>
#include <WiFiManager.h>
#include "config.h"

class SystemWiFiManager {
public:
    SystemWiFiManager();
    bool initWiFi();
    void processWiFi();
    bool isConnected();
    String getIP();

private:
    WiFiManager wm;
    bool connected;
};

extern SystemWiFiManager wifiManagerService;

#endif // WIFI_MANAGER_H
