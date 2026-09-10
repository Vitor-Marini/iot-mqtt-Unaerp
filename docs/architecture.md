# Arquitetura

## Visão geral

Simulação de uma estação meteorológica IoT. Um ESP32 com sensor BMP280 lê
temperatura, pressão e altitude e publica via MQTT. Um subscriber em Go grava
essas mensagens no InfluxDB, que o Grafana consulta para os painéis.

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
│  MQTT → InfluxDB   │  Decodifica o JSON, separa em channels e
└─────────┬──────────┘  escreve os pontos
          │
          │ HTTP write (line protocol, API v2)
          ▼
┌────────────────────┐
│  influxdb          │  Docker
│  TSDB              │  Measurements telemetry e healthcheck
└─────────┬──────────┘
          │
          │ Flux via HTTP :8086
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

## O subscriber por dentro

O subscriber é um processo de três camadas, ligadas por channels:

```
      MQTT
        │
        ▼
  messageHandler          roteia por sufixo do tópico e decodifica o JSON
      /       \
     ▼         ▼
telemetryChan  healthcheckChan
     │         │
     ▼         ▼
ProcessTelemetry  ProcessHealthcheck    goroutines que montam e gravam pontos
     └────┬────┘
          ▼
      InfluxDB
```

Os channels existem para desacoplar recepção de escrita: o callback do MQTT
devolve o controle imediatamente e não fica preso esperando o banco responder.

## Por que escrita direta, e não scrape

A versão anterior deste projeto usava Prometheus: o subscriber guardava o
último valor de cada sensor em memória, expunha `/metrics`, e o Prometheus
raspava a cada 15 s. **Migramos para InfluxDB com escrita direta.** Os motivos:

- **Nenhuma leitura é perdida.** No modelo de scrape, se o ESP32 publica a cada
  5 s e o Prometheus raspa a cada 15 s, duas de cada três leituras somem — o
  gauge só guarda a última. Escrevendo direto, toda mensagem publicada vira um
  ponto.
- **O timestamp é o do dispositivo.** O ponto é gravado com o `timestamp` do
  payload, não com o instante do scrape. A série reflete quando a medição
  aconteceu.
- **O modelo de dados casa com o payload.** `sensor_id` e `sensor_model` viram
  tags, as grandezas viram fields. Não é preciso inventar um nome de métrica
  por campo nem converter `status: "OK"` em série numérica só para caber no
  formato do Prometheus.

O que se perde, e como compensamos:

| Perda | Compensação |
|---|---|
| `up{job="subscriber"}` de graça — o scrape era o health check | O subscriber não serve HTTP; a saúde dele se vê pelo `docker compose ps` e pelos logs |
| Detecção de queda do ESP32 pelo `absent()` do PromQL | Consulta Flux sobre o último contato de cada `sensor_id` — ver abaixo |
| Familiaridade com PromQL | Consultas Flux prontas em [`services/influxdb/README.md`](../services/influxdb/README.md#consultas-úteis-flux) |
| Um serviço sem credencial | O InfluxDB exige token, que precisa estar igual em três `.env` |

## Topologia de máquinas

Cada serviço foi desenhado para rodar em uma máquina separada. Nenhum serviço
depende de outro estar no mesmo host: toda referência cruzada é um endereço
`host:porta` configurável por variável de ambiente.

| Serviço | Porta exposta | Precisa alcançar |
|---|---|---|
| `esp32-firmware` | — | `mosquitto:1883` |
| `mosquitto` | `1883` | — |
| `subscriber` | — (nenhuma) | `mosquitto:1883`, `influxdb:8086` |
| `influxdb` | `8086` | — |
| `grafana` | `3000` | `influxdb:8086` |

O `subscriber` é o único serviço que não abre porta nenhuma: ele só faz
conexões de saída. Isso simplifica o firewall — não há nada para liberar
chegando nele.

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
5. `messageHandler` decodifica em `models.Telemetry` e envia para
   `telemetryChan`.
6. `ProcessTelemetry` monta um ponto na measurement `telemetry`, com as tags
   `sensor_id`/`sensor_model`, os fields `temperature`/`pressure`/`altitude` e o
   timestamp do payload.
7. `WritePoint` grava no InfluxDB de forma síncrona.
8. O Grafana consulta via Flux e desenha.

## Detecção de queda

O firmware **não registra LWT**, então o broker não anuncia a saída de um
dispositivo. E como o subscriber não guarda estado em memória, não há nenhum
indicador de disponibilidade para zerar quando o silêncio começa.

A detecção é por **ausência, consultada no banco**: quanto tempo passou desde o
último ponto de cada `sensor_id`. A consulta Flux está em
[`services/influxdb/README.md`](../services/influxdb/README.md#consultas-úteis-flux)
e serve de base para um painel e, se quiserem, um alerta do Grafana.

O campo `status` do health-check é ortogonal a isso: ele diz se o **sensor**
respondeu (`OK`/`ERROR`), não se o dispositivo está no ar. Um ESP32 sem o BMP280
continua publicando, com `status: "ERROR"` e leituras zeradas.

## Decisões de projeto

| Decisão | Alternativa descartada | Motivo |
|---|---|---|
| InfluxDB com escrita direta | Prometheus com scrape de `/metrics` | Guarda toda leitura, com o timestamp do dispositivo; ver acima |
| Um `docker-compose.yml` por serviço | Um compose único | Cada serviço roda numa máquina separada; o compose da raiz é só para dev |
| Tópico por dispositivo (`devices/{MAC}/…`) | Tópico único `esp32/telemetry` | Permite filtrar um nó sem varrer o fluxo e habilita comando por difusão |
| Contrato em `docs/mqtt-contract.md` | Contrato implícito no código | Firmware (C++) e subscriber (Go) não compartilham tipos; o documento é o acoplamento |
| Token do InfluxDB fixado no `.env` | Deixar o InfluxDB gerar um aleatório | Sem token conhecido, o subscriber não sobe sem intervenção manual |
| Channels entre recepção e escrita | Gravar dentro do callback MQTT | O callback não pode ficar preso esperando o banco |
| Fila FreeRTOS entre sensor e rede | Ler e publicar no mesmo laço | A amostragem não pode travar em reconexão de MQTT |
