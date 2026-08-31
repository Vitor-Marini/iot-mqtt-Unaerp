# subscriber

Ponte entre o mundo MQTT e o mundo Prometheus. Assina os tópicos da estação,
valida o payload e expõe as leituras como métricas em `/metrics`.

> **Estado:** esqueleto organizado. Os pacotes, o Dockerfile e a configuração
> estão no lugar; a lógica ainda não foi escrita (ver [Implementação](#implementação)).

## O que faz

- Assina `/telemetry` e `/health-check` no broker (QoS 1), com reconexão
  automática.
- Valida cada payload contra
  [`docs/mqtt-contract.md`](../../docs/mqtt-contract.md) e descarta o que
  estiver fora do contrato, contabilizando o motivo.
- Mantém em memória o último valor de cada sensor e o expõe em
  `:2112/metrics` no formato do Prometheus.
- Marca o sensor como offline quando recebe `status: "offline"` (inclusive o
  LWT do ESP32) ou quando passa `SENSOR_STALE_AFTER` sem mensagem válida.

### Por que pull e não push

O subscriber **não escreve** no Prometheus — é o Prometheus que faz scrape
dele. Isso evita um componente extra (Pushgateway), faz do scrape um health
check natural (`up{job="subscriber"}`) e desacopla a taxa de publicação do
ESP32 da taxa de coleta. Detalhes em
[`docs/architecture.md`](../../docs/architecture.md#por-que-pull-e-não-push).

## Interface

| | |
|---|---|
| **Entrada** | MQTT `/telemetry` e `/health-check` |
| **Saída** | HTTP `GET :2112/metrics` (texto Prometheus) |
| **Depende de** | `mosquitto:1883` |
| **Consumido por** | `prometheus` |

### Métricas expostas

| Métrica | Tipo | Labels | Descrição |
|---|---|---|---|
| `weather_temperature_celsius` | gauge | `sensor_id`, `sensor_model` | Última temperatura lida |
| `weather_pressure_hpa` | gauge | `sensor_id`, `sensor_model` | Última pressão barométrica |
| `weather_altitude_meters` | gauge | `sensor_id`, `sensor_model` | Última altitude estimada |
| `weather_sensor_up` | gauge | `sensor_id` | `1` online, `0` offline ou sem dados recentes |
| `weather_sensor_last_seen_timestamp_seconds` | gauge | `sensor_id` | Unix time da última mensagem válida |
| `weather_sensor_uptime_seconds` | gauge | `sensor_id` | Uptime relatado pelo ESP32 |
| `weather_sensor_wifi_rssi_dbm` | gauge | `sensor_id` | Sinal WiFi relatado pelo ESP32 |
| `weather_messages_received_total` | counter | `topic` | Mensagens recebidas |
| `weather_messages_invalid_total` | counter | `topic`, `reason` | Mensagens descartadas |
| `weather_broker_connected` | gauge | — | `1` se a conexão MQTT está ativa |

`reason` é um conjunto fechado (`malformed_json`, `missing_field`,
`out_of_range`, `bad_timestamp`, `unknown`) — label de cardinalidade alta
derruba o Prometheus.

## Estrutura

```
subscriber/
├── cmd/subscriber/main.go     # composição: lê config, liga as peças, sobe o HTTP
├── internal/
│   ├── config/                # carrega e valida as variáveis de ambiente
│   ├── mqtt/                  # cliente MQTT: conexão, reconexão, assinaturas
│   ├── telemetry/             # parsing e validação dos payloads do contrato
│   └── metrics/               # registro e atualização das métricas Prometheus
├── Dockerfile                 # multi-stage: build Go → alpine, usuário não-root
├── docker-compose.yml
├── Makefile
└── .env.example
```

O layout `cmd/` + `internal/` é a convenção Go: `cmd/` só faz composição,
`internal/` guarda os pacotes que não devem ser importados de fora do módulo.
Cada pacote tem uma responsabilidade e um `doc.go` explicando-a.

## Pré-requisitos

- Go 1.21+ (para desenvolvimento local).
- Docker Engine 20.10+ e Docker Compose v2 (para rodar em container).
- Um broker `mosquitto` alcançável.

## Configuração

Toda por variável de ambiente — nada de arquivo de config no container.

```bash
cp .env.example .env
```

| Variável | Descrição | Padrão |
|---|---|---|
| `MQTT_BROKER_URL` | URL do broker (`tcp://host:porta`) | `tcp://mosquitto:1883` |
| `MQTT_CLIENT_ID` | Client ID no broker (precisa ser único) | `subscriber-estacao` |
| `MQTT_USERNAME` / `MQTT_PASSWORD` | Vazios para conexão anônima | `""` |
| `MQTT_TELEMETRY_TOPIC` | Tópico de telemetria | `/telemetry` |
| `MQTT_HEALTH_TOPIC` | Tópico de health check | `/health-check` |
| `MQTT_QOS` | QoS das assinaturas (`0`–`2`) | `1` |
| `METRICS_ADDR` | Endereço do servidor HTTP | `:2112` |
| `METRICS_PATH` | Caminho das métricas | `/metrics` |
| `SUBSCRIBER_PORT` | Porta publicada no host pelo Compose | `2112` |
| `SENSOR_STALE_AFTER` | Silêncio até marcar o sensor offline | `90s` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |

## Como rodar

### Com Docker (recomendado)

```bash
cp .env.example .env      # ajuste MQTT_BROKER_URL para o IP do broker
docker compose up -d --build
docker compose logs -f
```

### Local, sem container

```bash
export MQTT_BROKER_URL=tcp://localhost:1883
make run
```

Outros alvos: `make help`.

## Como validar

```bash
# O endpoint responde?
curl -s http://localhost:2112/metrics | head

# As métricas da estação estão presentes?
curl -s http://localhost:2112/metrics | grep '^weather_'
```

Com o broker no ar, publique uma leitura sintética e veja o gauge mudar:

```bash
mosquitto_pub -h localhost -t '/telemetry' -m \
  '{"sensor_id":"teste","sensor_model":"BMP280","temperature":25.1,"pressure":1013.2,"altitude":120.5,"timestamp":"2026-08-31T14:03:21Z"}'

curl -s http://localhost:2112/metrics | grep weather_temperature_celsius
# weather_temperature_celsius{sensor_id="teste",sensor_model="BMP280"} 25.1
```

Testes unitários (não precisam de broker):

```bash
make test
```

## Implementação

O esqueleto está pronto; falta escrever a lógica. Dependências previstas:

```bash
go get github.com/eclipse/paho.mqtt.golang
go get github.com/prometheus/client_golang/prometheus
go mod tidy
```

Ordem sugerida, do núcleo testável para fora:

1. `internal/telemetry` — structs do contrato e validação. O `timestamp`
   precisa aceitar as duas formas do contrato (RFC 3339 e unix seconds).
   É o pacote mais fácil de testar: só bytes entrando e structs saindo.
2. `internal/metrics` — registro dos coletores e métodos de atualização.
   Testável com `prometheus/testutil`.
3. `internal/config` — leitura das variáveis da tabela acima, com defaults.
4. `internal/mqtt` — conexão e assinaturas, delegando o payload aos pacotes
   acima.
5. `cmd/subscriber/main.go` — composição, servidor HTTP e tratamento de
   `SIGINT`/`SIGTERM`.

## Problemas comuns

| Sintoma | Causa provável |
|---|---|
| `/metrics` responde mas sem `weather_*` | Nenhuma mensagem chegou ainda — gauges só aparecem depois da primeira leitura |
| `connection refused` no boot | `MQTT_BROKER_URL` aponta para host errado, ou o broker ainda não subiu |
| Sensor sempre `weather_sensor_up 0` | `SENSOR_STALE_AFTER` menor que o `PUBLISH_INTERVAL_MS` do firmware |
| Cliente derrubado toda hora | Dois processos com o mesmo `MQTT_CLIENT_ID` — o broker expulsa o anterior |
| `weather_messages_invalid_total` subindo | Payload fora do contrato; o label `reason` diz qual regra falhou |
