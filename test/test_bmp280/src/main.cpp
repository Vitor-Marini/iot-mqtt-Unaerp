#include <Arduino.h>
#include <Wire.h>
#include <Adafruit_BMP280.h>

// Definicao dos pinos I2C customizados para o ESP32
#define I2C_SDA 21
#define I2C_SCL 22

// Instancia da biblioteca do sensor BMP280
Adafruit_BMP280 bmp;

void setup() {
    // 1. Inicializacao da comunicação Serial
    Serial.begin(115200);
    while (!Serial && millis() < 3000); // Aguarda a conexao serial por ate 3s

    Serial.println("\n==============================================");
    Serial.println("  ESP32 - Teste de Diagnostico do Sensor BMP280");
    Serial.println("==============================================\n");

    // 2. Inicializacao do barramento I2C com pinos especificos (SDA=21, SCL=22)
    Serial.printf("[INFO] Inicializando barramento I2C (SDA: GPIO %d, SCL: GPIO %d)...\n", I2C_SDA, I2C_SCL);
    Wire.begin(I2C_SDA, I2C_SCL);

    // 3. Tentativa de inicializacao do sensor nos enderecos I2C padrao (0x76 ou 0x77)
    bool status = bmp.begin(0x76);
    if (!status) {
        // Tenta o endereco alternativo 0x77 se 0x76 falhar
        status = bmp.begin(0x77);
    }

    // 4. Verificacao de integridade e leitura do Chip ID
    if (!status) {
        Serial.println("\n[ERRO CRITICO] Falha ao detectar o sensor BMP280!");
        Serial.println("Verifique:");
        Serial.println("  1. Se as conexoes SDA (GPIO 21) e SCL (GPIO 22) estao corretas.");
        Serial.println("  2. Se a alimentacao VCC (3.3V) e GND estao ligadas firmemente.");
        Serial.println("  3. Se o endereco I2C do modulo eh 0x76 ou 0x77.");
        Serial.println("\nExecucao interrompida.");
        
        // Interrompe a execucao em loop infinito
        while (1) {
            delay(1000);
        }
    }

    // Leitura do registrador de Chip ID
    uint8_t chipID = bmp.sensorID();
    Serial.println("[SUCESSO] Sensor BMP280 detectado no barramento I2C!");
    Serial.printf("[DIAGNOSTICO] Chip ID lido: 0x%02X (Esperado: 0x58 para BMP280)\n\n", chipID);

    // Configuracoes de amostragem padrao do BMP280
    bmp.setSampling(Adafruit_BMP280::MODE_NORMAL,     /* Modo de Operacao */
                    Adafruit_BMP280::SAMPLING_X2,     /* Temp. oversampling */
                    Adafruit_BMP280::SAMPLING_X16,    /* Pressao oversampling */
                    Adafruit_BMP280::FILTER_X16,      /* Filtragem IIR */
                    Adafruit_BMP280::STANDBY_MS_500); /* Tempo de Standby */

    Serial.println("Iniciando leituras continuas a cada 2 segundos...\n");
}

void loop() {
    // Leitura das grandezas fisicas
    float temperatura = bmp.readTemperature();          // Em °C
    float pressao = bmp.readPressure() / 100.0F;        // Converte de Pa para hPa
    float altitude = bmp.readAltitude(1013.25);         // Altitude calculada com pressao de referencia ao nivel do mar (1013.25 hPa)

    // Impressao limpa e formatada no terminal
    Serial.println("----------------------------------------------");
    Serial.printf(" Temperatura : %.2f °C\n", temperatura);
    Serial.printf(" Pressao     : %.2f hPa\n", pressao);
    Serial.printf(" Altitude Est: %.2f m\n", altitude);
    Serial.println("----------------------------------------------\n");

    // Intervalo de 2 segundos entre leituras
    delay(2000);
}
