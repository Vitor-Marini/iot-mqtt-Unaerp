# Arquitetura

## Visão geral

Simulação de uma estação meteorológica IoT. Um ESP32 com sensor BMP280 lê
temperatura, pressão e altitude e publica via MQTT. Um subscriber em Go traduz
essas mensagens em métricas Prometheus, que o Grafana consulta para os painéis.

```
┌────────────────────┐
│  esp32-firmware    │  Hardware físico
│  ESP32 + BMP280    │  Lê o sensor via I²C, conecta no WiFi,
└─────────┬──────────┘  publica JSON a cada 5 s
          │
          │ MQTT publish  /telemetry  /health-check
          │ TCP 1883
          ▼
┌────────────────────┐
│  mosquitto         │  Docker
│  MQTT broker       │  Roteia mensagens, guarda LWT e retained
└─────────┬──────────┘
          │
          │ MQTT subscribe  (QoS 1)
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
│  Visualização      │  Dashboard "Estação Meteorológica" provisionado
└────────────────────┘
```

## Por que pull e não push

O subscriber **não escreve** no Prometheus. Ele mantém o último valor de cada
sensor em memória e expõe em `/metrics`; o Prometheus faz scrape. Esse é o
modelo idiomático do Prometheus e traz três vantagens neste projeto:

- **Sem componente extra.** Pushgateway ou `remote_write` adicionariam um
  serviço ou uma flag a mais para manter.
- **O scrape é o health check.** Se o subscriber cair, o Prometheus registra
  `up{job="subscriber"} == 0` sem nenhum código adicional.
- **Desacoplamento da taxa.** O ESP32 pode publicar a 5 s e o Prometheus
  raspar a 15 s sem perder consistência — o gauge sempre tem o último valor.

O custo é que rajadas mais rápidas que o intervalo de scrape são achatadas
(só o último valor entre dois scrapes vira ponto na série). Para uma estação
meteorológica, cujas grandezas mudam devagar, isso é irrelevante.

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

1. O ESP32 lê o BMP280 via I²C (endereço `0x76`).
2. Serializa o payload com ArduinoJson conforme
   [mqtt-contract.md](mqtt-contract.md).
3. Publica em `/telemetry` com QoS 1.
4. O Mosquitto entrega ao subscriber, que tem uma assinatura ativa.
5. O subscriber faz `json.Unmarshal`, valida faixas e descarta o que estiver
   fora do contrato (contabilizando em `weather_messages_invalid_total`).
6. Em caso válido, atualiza `weather_temperature_celsius`,
   `weather_pressure_hpa` e `weather_altitude_meters` com o label `sensor_id`,
   e registra o instante em `weather_sensor_last_seen_timestamp_seconds`.
7. No próximo scrape, o Prometheus lê `/metrics` e grava os pontos.
8. O Grafana consulta via PromQL e desenha.

## Decisões de projeto

| Decisão | Alternativa descartada | Motivo |
|---|---|---|
| Métricas por pull (`/metrics`) | Pushgateway, `remote_write` | Menos peças móveis; ver acima |
| Um `docker-compose.yml` por serviço | Um compose único | Cada serviço roda numa máquina separada; o compose da raiz é só para dev |
| Contrato em `docs/mqtt-contract.md` | Contrato implícito no código | Firmware (C++) e subscriber (Go) não compartilham tipos; o documento é o acoplamento |
| `file_sd_configs` no Prometheus | `static_configs` | Trocar o IP do subscriber não exige reiniciar o Prometheus |
| Faixas validadas no subscriber | Confiar no firmware | Um sensor com defeito publica `-140 °C`; isso não deve virar série temporal |
