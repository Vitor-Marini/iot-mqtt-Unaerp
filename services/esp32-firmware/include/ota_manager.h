#ifndef OTA_MANAGER_H
#define OTA_MANAGER_H

#include <Arduino.h>
#include <ArduinoOTA.h>
#include <HTTPClient.h>
#include <HTTPUpdate.h>
#include <Update.h>
#include <esp_ota_ops.h>
#include "config.h"

struct OTACommandRequest {
    char url[256];
    char version[32];
    char md5[34];
};

class SystemOTAManager {
public:
    SystemOTAManager();
    void init();
    void handle();

    /**
     * @brief Executa atualizacao remota via HTTP baixando o binario informado na URL.
     * @param url URL completa contendo o binario (ex: http://192.168.1.100:8080/firmware/firmware.bin)
     * @param version Versao sendo instalada
     * @param expectedMd5 Checksum MD5 esperado (opcional)
     * @return bool True se a atualizacao for iniciada com sucesso.
     */
    bool performHTTPUpdate(const String& url, const String& version, const String& expectedMd5 = "");

    /**
     * @brief Valida o firmware atual no bootloader para cancelar rollback automatico.
     * Deve ser chamada quando WiFi e MQTT estiverem conectados e saudaveis.
     */
    void confirmFirmwareValidity();

    bool isUpdating() const { return updating; }

private:
    bool initialized;
    bool updating;
};

extern SystemOTAManager otaService;
extern QueueHandle_t xOTAQueue;

#endif // OTA_MANAGER_H
