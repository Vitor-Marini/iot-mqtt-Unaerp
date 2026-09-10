# subscriber

Ponte entre o mundo MQTT e o banco temporal. Assina os tópicos da estação,
decodifica o JSON e grava os pontos no InfluxDB.

Escrito em Go. **Não serve HTTP** — só faz conexões de saída, uma para o broker
e uma para o InfluxDB.

## O que faz

- Assina `devices/+/telemetry` e `devices/+/health-check` em QoS 0. O wildcard é
  necessário porque o MAC do dispositivo faz parte do tópico.
- Decodifica cada payload nos tipos de [`models/`](models/) e envia para um
  channel por tipo de mensagem.
- Duas goroutines (`ProcessTelemetry` e `ProcessHealthcheck`) consomem esses
  channels e escrevem no InfluxDB. Assim o recebimento MQTT fica desacoplado da
  latência de escrita no banco.
- Converte `status: "OK"` em `1` e qualquer outro valor em `0`, porque o
  InfluxDB guarda números, não texto de estado.

O formato das mensagens está em
[`docs/mqtt-contract.md`](../../docs/mqtt-contract.md); a fonte da verdade é o
firmware.

## Interface

| | |
|---|---|
| **Entrada** | MQTT `devices/+/telemetry` e `devices/+/health-check` |
| **Saída** | escritas HTTP no InfluxDB (line protocol, API v2) |
| **Depende de** | `mosquitto:1883` e `influxdb:8086` |
| **Consumido por** | ninguém diretamente — o Grafana lê o InfluxDB |

### Dados gravados

Duas *measurements*, ambas com as tags `sensor_id` e `sensor_model`:

| Measurement | Fields | Origem |
|---|---|---|
| `telemetry` | `temperature`, `pressure`, `altitude` | `devices/+/telemetry` |
| `healthcheck` | `status`, `rssi`, `free_heap`, `uptime_ms` | `devices/+/health-check` |

As tags viram *séries* separadas, então vários ESP32 convivem no mesmo bucket
sem se misturar. Consultas Flux prontas em
[`../influxdb/README.md`](../influxdb/README.md#consultas-úteis-flux).

## Estrutura

Pacote único na raiz do módulo, mais `models/`:

| Arquivo | Responsabilidade |
|---|---|
| `main.go` | Carrega o `.env`, abre o InfluxDB, configura o cliente MQTT e assina os tópicos |
| `mqtt.go` | `messageHandler`: roteia por sufixo do tópico, decodifica o JSON e publica no channel |
| `workers.go` | As duas goroutines que consomem os channels e montam os pontos |
| `server.go` | Cliente InfluxDB (`InitInfluxDB` / `CloseInfluxDB`) |
| `models/telemetry.go` | Struct do payload de telemetria |
| `models/healthcheck.go` | Struct do payload de health check |
| `mock_esp32.sh` | Publica telemetria falsa no broker, para testar sem hardware |

> O nome `server.go` é herança da fase anterior, quando o arquivo servia o
> endpoint `/metrics`. Hoje ele guarda o cliente do InfluxDB.

## Pré-requisitos

- **Go 1.26+** para desenvolvimento local (`go.mod` declara `go 1.26.5`).
- Docker Engine 20.10+ e Docker Compose v2 para rodar em container.
- Um `mosquitto` e um `influxdb` alcançáveis.

## Configuração

Toda por variável de ambiente. O código lê o `.env` do diretório atual via
`godotenv` e aceita as mesmas chaves como variáveis de ambiente — é assim que o
Docker as injeta.

```bash
cp .env.example .env
```

| Variável | Descrição | Padrão no código |
|---|---|---|
| `MQTT_HOST` | Host do broker | `localhost` |
| `MQTT_PORT` | Porta do broker | `1883` |
| `SUBSCRIBER_ID` | Client ID no broker (precisa ser único) | `subscriber` |
| `INFLUX_HOST` | URL do InfluxDB (`http://host:8086`) | **nenhum** |
| `TOKEN_INFLUX` | Token de escrita | **nenhum** |
| `INFLUX_ORG` | Organização | **nenhum** |
| `INFLUX_BUCKET` | Bucket | **nenhum** |

As quatro variáveis do InfluxDB **não têm valor padrão**: sem elas o serviço
sobe, conecta no MQTT e falha em toda escrita. Por isso o
`docker-compose.yml` deste serviço exige o `.env`.

Atenção ao nome: é `TOKEN_INFLUX`, não `INFLUX_TOKEN`. O valor é o mesmo
`INFLUXDB_TOKEN` de [`services/influxdb/.env`](../influxdb/.env.example).

## Como rodar

### Com Docker, em máquina própria

```bash
cp .env.example .env
# ajuste MQTT_HOST, INFLUX_HOST e TOKEN_INFLUX para os IPs/token reais
docker compose up -d --build
docker compose logs -f
```

### Junto com a stack inteira

Não use o compose deste diretório: use o da raiz, que já liga tudo por DNS
interno. Ver [README da raiz](../../README.md#execução).

### Local, sem container

```bash
cp .env.example .env    # com MQTT_HOST=localhost e INFLUX_HOST=http://localhost:8086
make run
```

Outros alvos: `make help`.

## Como validar

Com o broker e o InfluxDB no ar, gere tráfego falso:

```bash
make mock        # ou: ./mock_esp32.sh
```

O log do subscriber deve mostrar cada mensagem chegando:

```
TOPICO: devices/A1B2C3D4E5F6/telemetry
PAYLOAD: {"sensor_id":"A1B2C3D4E5F6",...}
Telemetry recebida do dispositivo A1B2C3D4E5F6
========== TELEMETRY ==========
Sensor ID: A1B2C3D4E5F6
...
```

Confirme que os pontos chegaram ao banco:

```bash
docker compose -f ../influxdb/docker-compose.yml exec influxdb \
  influx query --org esp32 --token "$INFLUXDB_TOKEN" '
from(bucket: "sensors")
  |> range(start: -5m)
  |> filter(fn: (r) => r._measurement == "telemetry")
  |> last()
'
```

Uma mensagem única, sem o mock:

```bash
mosquitto_pub -h localhost -t "devices/A1B2C3D4E5F6/telemetry" -m \
  '{"sensor_id":"A1B2C3D4E5F6","sensor_model":"BMP280","temperature":24.5,"pressure":1013.25,"altitude":540.2,"timestamp":1787960400}'
```

## Comportamentos conhecidos

Coisas que valem saber antes de depurar:

- **Escrita síncrona.** Usa `WriteAPIBlocking`, ou seja, uma requisição HTTP por
  ponto. Simples e com erro visível na hora, mas se o InfluxDB ficar lento a
  goroutine correspondente segura o channel. Para volume maior, o caminho é a
  API assíncrona — está anotado no fim de `server.go`.
- **Pontos em 1970.** O timestamp do ponto vem do payload
  (`time.Unix(telemetry.Timestamp, 0)`). Se o ESP32 não sincronizou NTP, ele
  envia o uptime em segundos, e o ponto cai perto da época Unix. Ver o
  [contrato](../../docs/mqtt-contract.md#timestamp-é-sempre-um-número).
- **Payload não é validado.** O subscriber decodifica e grava. Um BMP280 ausente
  faz o firmware publicar zeros, e esses zeros entram no banco. Filtre no
  Grafana, ou trate na origem.
- **`deviceID` do tópico é só logado.** O `sensor_id` gravado como tag vem do
  payload, não do segmento do tópico.

## Problemas comuns

| Sintoma | Causa provável |
|---|---|
| `Tópico desconhecido` no log | Sufixo fora de `/telemetry` e `/health-check` |
| Nada chega, mas o ESP32 publica | `mqtt_topic_base` do firmware diferente de `devices` |
| `unauthorized: unauthorized access` | `TOKEN_INFLUX` diferente do token do InfluxDB |
| `bucket not found` | `INFLUX_BUCKET` diferente do bucket provisionado |
| Conecta no MQTT mas nada aparece no banco | `INFLUX_HOST` vazio ou inalcançável — as 4 vars do Influx são obrigatórias |
| Cliente derrubado toda hora | Dois processos com o mesmo `SUBSCRIBER_ID` |
| Dados antigos aparecem ao (re)iniciar | O firmware publica com retain; ver o contrato |
| `go: go.mod requires go >= 1.26.5` | Go local mais antigo — use Docker ou atualize |
