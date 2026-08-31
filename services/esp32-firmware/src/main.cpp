// Estação Meteorológica — firmware ESP32 + BMP280
//
// Ponto de entrada do firmware. O que precisa ser implementado aqui está
// descrito no README deste serviço e no contrato em docs/mqtt-contract.md.
//
// Responsabilidades:
//   setup()  — Serial, I2C/BMP280, WiFi, NTP e configuração do cliente MQTT
//              (incluindo o LWT em /health-check).
//   loop()   — manter WiFi/MQTT conectados, ler o sensor e publicar
//              /telemetry a cada PUBLISH_INTERVAL_MS e
//              /health-check a cada HEALTH_INTERVAL_MS.

#include <Arduino.h>
#include "config.h"

void setup() {
  Serial.begin(115200);
  // TODO: inicializar BMP280 (I2C), WiFi, NTP e o cliente MQTT.
}

void loop() {
  // TODO: reconectar se necessário, ler o sensor e publicar.
}
