#include "ota_manager.h"
#include "mqtt_client.h"

SystemOTAManager otaService;

SystemOTAManager::SystemOTAManager() : initialized(false), updating(false) {}

void SystemOTAManager::init() {
    String hostname = "ESP32-Node-" + getDeviceMacId();
    ArduinoOTA.setHostname(hostname.c_str());
    ArduinoOTA.setPassword("unaerp123");

    ArduinoOTA.onStart([]() {
        String type = (ArduinoOTA.getCommand() == U_FLASH) ? "sketch" : "filesystem";
        Serial.println("\n[OTA Local] Iniciando gravacao via porta local (" + type + ")...");
    });

    ArduinoOTA.onEnd([]() {
        Serial.println("\n[OTA Local] Atualizacao concluida! Reiniciando...");
    });

    ArduinoOTA.onProgress([](unsigned int progress, unsigned int total) {
        Serial.printf("[OTA Local] Progresso: %u%%\r", (progress / (total / 100)));
    });

    ArduinoOTA.onError([](ota_error_t error) {
        Serial.printf("[OTA Local Erro %u]: ", error);
        if (error == OTA_AUTH_ERROR) Serial.println("Falha de Autenticacao");
        else if (error == OTA_BEGIN_ERROR) Serial.println("Falha ao Iniciar");
        else if (error == OTA_CONNECT_ERROR) Serial.println("Falha na Conexao");
        else if (error == OTA_RECEIVE_ERROR) Serial.println("Falha na Recepcao");
        else if (error == OTA_END_ERROR) Serial.println("Falha ao Finalizar");
    });

    ArduinoOTA.begin();
    initialized = true;
    Serial.println("[OTA] Servico ArduinoOTA local ativo. Hostname: " + hostname);
}

void SystemOTAManager::handle() {
    if (initialized && !updating) {
        ArduinoOTA.handle();
    }
}

bool SystemOTAManager::performHTTPUpdate(const String& url, const String& version, const String& expectedMd5) {
    if (url.length() == 0) return false;

    updating = true;
    Serial.println("\n==================================================");
    Serial.println("           INICIANDO ATUALIZAÇÃO HTTP OTA         ");
    Serial.println("==================================================");
    Serial.println(" [OTA] URL     : " + url);
    Serial.println(" [OTA] Versao  : " + version);
    if (expectedMd5.length() > 0) {
        Serial.println(" [OTA] MD5 Exp : " + expectedMd5);
    }
    Serial.println("==================================================");

    // Notifica o broker de que o download foi iniciado
    mqttService.publishOTAStatus("DOWNLOADING", version, "Iniciando download do novo firmware via HTTP...");
    vTaskDelay(pdMS_TO_TICKS(300)); // Aguarda envio do pacote MQTT

    // Callback de progresso do Update para alimentar o Task Watchdog (TWDT)
    // e ceder CPU para a tarefa IDLE0 do Core 0 durante o download de arquivos grandes
    Update.onProgress([](size_t current, size_t total) {
        static uint32_t lastYield = 0;
        uint32_t now = millis();
        if (now - lastYield >= 50) {
            lastYield = now;
            // Cede 2ms para a tarefa IDLE0 rodar e resetar o Watchdog Timer
            vTaskDelay(pdMS_TO_TICKS(2));
        }
    });

    WiFiClient client;
    client.setTimeout(30); // Timeout ampliado para redes com concorrencia / baixa velocidade
    httpUpdate.rebootOnUpdate(true);
    
    if (expectedMd5.length() > 0) {
        Update.setMD5(expectedMd5.c_str());
    }

    t_httpUpdate_return ret = httpUpdate.update(client, url);

    // Se a funcao retornar, significa que o update falhou (em caso de sucesso a placa reinicia automaticamente)
    updating = false;
    String errStr = httpUpdate.getLastErrorString();
    int errCode = httpUpdate.getLastError();

    Serial.printf("[OTA ERROR] Falha na atualizacao (%d): %s\n", errCode, errStr.c_str());

    mqttService.publishOTAStatus("FAILED", version, "Falha durante download/gravacao do firmware.", 
                                 String(errCode) + ": " + errStr);

    return false;
}

void SystemOTAManager::confirmFirmwareValidity() {
    const esp_partition_t* running = esp_ota_get_running_partition();
    esp_ota_img_states_t ota_state;

    if (esp_ota_get_state_partition(running, &ota_state) == ESP_OK) {
        if (ota_state == ESP_OTA_IMG_PENDING_VERIFY) {
            Serial.println("\n[OTA Safety] ✅ Novo firmware iniciado com sucesso! Cancelando rollback automatico no bootloader...");
            esp_ota_mark_app_valid_cancel_rollback();

            // Notifica o servidor e o broker de que a atualizacao foi concluida com exito
            mqttService.publishOTAStatus("SUCCESS", FIRMWARE_VERSION, "Firmware validado com sucesso apos boot OTA.");
        }
    }
}
