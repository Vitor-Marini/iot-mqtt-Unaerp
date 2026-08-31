# Arquitetura

## Visão geral

Simulação de uma estação meteorológica IoT. Um ESP32 com sensor BMP280 lê
temperatura, pressão e altitude e publica via MQTT. Um subscriber em Go traduz
essas mensagens em métricas Prometheus, que o Grafana consulta para os painéis.

```
┌────────────────────┐
│  esp32-firmware    │  Hardware físico
│  ESP32 + BMP280    │  Duas tasks FreeRTOS: uma lê o sensor a cada 5 s
└─────────┬──────────┘  e enfileira, outra cuida de WiFi/MQTT e publica
          │
          │ MQTT publish (QoS 0)
          │   devices/{MAC}/telemetry
          │   devices/{MAC}/health-check
          ▼
┌────────────────────┐
│  mosquitto         │  Docker
│  MQTT broker       │  Roteia por dispositivo; guarda as retidas
└─────────┬──────────┘
          │
          │ MQTT subscribe  devices/+/telemetry
          │                 devices/+/health-check
          ▼
┌────────────────────┐
│  subscriber        │  Docker (Go)
│  MQTT → Prometheus │  Valida o payload, atualiza gauges/counters
└─────────┬──────────┘  e expõe HTTP :2112/metrics
          │
          │ HTTP scrape a cada 15 s
          ▼
┌────────────────────┐
│  prometheus        │  Docker
│  TSDB              │  Armazena a série temporal, retenção 15 d
└─────────┬──────────┘
          │
          │ PromQL via HTTP :9090
          ▼
┌────────────────────┐
│  grafana           │  Docker
│  Visualização      │  Painéis da estação
└────────────────────┘
```

O formato exato das mensagens está em [`mqtt-contract.md`](mqtt-contract.md),
que descreve o firmware — a fonte da verdade é o código em
[`services/esp32-firmware/`](../services/esp32-firmware/).

## O firmware por dentro

O ESP32 não roda um laço único. São duas tasks FreeRTOS fixadas no core 1,
desacopladas por uma fila:

| Task | Prioridade | O que faz |
|---|---|---|
| `vTaskSensors` | 1 | Lê o BMP280 a cada `SENSOR_READ_INTERVAL_MS` e enfileira um `SensorPayload` |
| `vTaskWiFiMQTT` | 2 | Mantém WiFi e MQTT, consome a fila e publica telemetria e health-check |

A fila (`xSensorQueue`, 10 posições) é o ponto importante: a amostragem do
sensor acontece em cadência determinística, sem travar em latência de rede ou
em reconexão de MQTT. Se a rede cair e a fila encher, a leitura mais nova é
descartada com um aviso no serial — a task do sensor nunca bloqueia.

A conexão WiFi tenta primeiro as credenciais do `secrets.ini`; falhando em
15 s, cai para o portal cativo do WiFiManager (`ESP32-Setup-Portal-{sufixo do
MAC}`, em `http://192.168.4.1`). Assim que conecta, sincroniza o relógio por
SNTP para poder carimbar os payloads em UTC.

## Por que pull e não push

O subscriber **não escreve** no Prometheus. Ele mantém o último valor de cada
sensor em memória e expõe em `/metrics`; o Prometheus faz scrape. Esse é o
modelo idiomático do Prometheus e traz três vantagens neste projeto:

- **Sem componente extra.** Pushgateway ou `remote_write` adicionariam um
  serviço ou uma flag a mais para manter.
- **O scrape é o health check.** Se o subscriber cair, o Prometheus registra
  `up{job="subscriber"} == 0` sem nenhum código adicional.
- **Desacoplamento da taxa.** O ESP32 publica a 5 s e o Prometheus raspa a
  15 s sem perder consistência — o gauge sempre tem o último valor.

O custo é que rajadas mais rápidas que o intervalo de scrape são achatadas.
Para uma estação meteorológica, cujas grandezas mudam devagar, isso é
irrelevante.

## Topologia de máquinas

Cada serviço foi desenhado para rodar em uma máquina separada. Nenhum serviço
depende de outro estar no mesmo host: toda referência cruzada é um endereço
`host:porta` configurável por variável de ambiente ou arquivo de alvos.

| Serviço | Porta exposta | Precisa alcançar |
|---|---|---|
| `esp32-firmware` | — | `mosquitto:1883` |
| `mosquitto` | `1883` | — |
| `subscriber` | `2112` | `mosquitto:1883` |
| `prometheus` | `9090` | `subscriber:2112` |
| `grafana` | `3000` | `prometheus:9090` |

O `docker-compose.yml` da raiz existe apenas para desenvolvimento: ele sobe os
quatro serviços Docker numa rede única, onde os nomes acima resolvem por DNS.
Ver [../README.md](../README.md#execução).

## Fluxo de dados de uma leitura

1. `vTaskSensors` acorda no intervalo e lê o BMP280 via I²C (`0x76`, com
   `0x77` como alternativa).
2. Empacota em `SensorPayload` e envia para `xSensorQueue`.
3. `vTaskWiFiMQTT` retira da fila, serializa com ArduinoJson conforme
   [mqtt-contract.md](mqtt-contract.md) e publica em
   `devices/{MAC}/telemetry`.
4. O Mosquitto entrega ao subscriber, que assina `devices/+/telemetry`.
5. O subscriber faz `json.Unmarshal`, valida faixas e descarta o que estiver
   fora do contrato (contabilizando em `weather_messages_invalid_total`).
6. Em caso válido, atualiza `weather_temperature_celsius`,
   `weather_pressure_hpa` e `weather_altitude_meters` com o label `sensor_id`,
   e registra o instante em `weather_sensor_last_seen_timestamp_seconds`.
7. No próximo scrape, o Prometheus lê `/metrics` e grava os pontos.
8. O Grafana consulta via PromQL e desenha.

## Detecção de queda

O firmware **não registra LWT**, então o broker não anuncia a saída de um
dispositivo. A detecção é por ausência: sem mensagem válida de um `sensor_id`
por mais de `SENSOR_STALE_AFTER` (padrão 90 s, o triplo do intervalo de
health-check), o subscriber zera `weather_sensor_up` daquele dispositivo.

O campo `status` do health-check é ortogonal a isso: ele diz se o **sensor**
respondeu (`"OK"`/`"ERROR"`), não se o dispositivo está no ar.

## Decisões de projeto

| Decisão | Alternativa descartada | Motivo |
|---|---|---|
| Métricas por pull (`/metrics`) | Pushgateway, `remote_write` | Menos peças móveis; ver acima |
| Um `docker-compose.yml` por serviço | Um compose único | Cada serviço roda numa máquina separada; o compose da raiz é só para dev |
| Tópico por dispositivo (`devices/{MAC}/…`) | Tópico único `/telemetry` | Permite filtrar um nó sem varrer o fluxo e habilita comando por difusão |
| Contrato em `docs/mqtt-contract.md` | Contrato implícito no código | Firmware (C++) e subscriber (Go) não compartilham tipos; o documento é o acoplamento |
| `file_sd_configs` no Prometheus | `static_configs` | Trocar o IP do subscriber não exige reiniciar o Prometheus |
| Faixas validadas no subscriber | Confiar no firmware | Sem o BMP280, o firmware publica zeros; isso não deve virar série temporal |
| Fila FreeRTOS entre sensor e rede | Ler e publicar no mesmo laço | A amostragem não pode travar em reconexão de MQTT |
