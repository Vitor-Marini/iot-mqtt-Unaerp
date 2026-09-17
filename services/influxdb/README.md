# influxdb

Banco de dados temporal do projeto. Recebe as escritas do `subscriber` e
responde às consultas Flux do Grafana.

Imagem: [`influxdb:2.7`](https://hub.docker.com/_/influxdb).

## O que faz

- Guarda duas *measurements*, escritas pelo subscriber:
  - `telemetry` — `temperature`, `pressure`, `altitude`
  - `healthcheck` — `status`, `version`, `ip`, `rssi`, `free_heap`, `uptime_ms`
- Ambas com as tags `device_id` (o MAC), `device_name` (nome legível) e
  `sensor_model`, o que permite separar vários ESP32 no mesmo bucket.
- Em `healthcheck`, `version` e `ip` são **fields**, não tags: mudam no tempo
  (a cada OTA e a cada lease de DHCP) e como tags fraturariam as séries.
- Se autoprovisiona no primeiro boot: cria a organização, o bucket, o usuário
  admin e o token, sem nenhum passo manual.
- Expõe a UI e a API HTTP em `:8086`.

## Interface

| | |
|---|---|
| **Entrada** | escritas HTTP do `subscriber` (line protocol, API v2) |
| **Saída** | consultas Flux em `:8086` + UI web |
| **Depende de** | nada |
| **Consumido por** | `subscriber` (escreve), `grafana` (lê) |

## Pré-requisitos

- Docker Engine 20.10+ e Docker Compose v2.
- Porta `8086` liberada no firewall — o subscriber e o Grafana vêm de outras
  máquinas.

## Configuração

```bash
cp .env.example .env
```

| Variável | Descrição | Padrão |
|---|---|---|
| `INFLUXDB_PORT` | Porta publicada no host | `8086` |
| `INFLUXDB_USERNAME` | Usuário admin da UI | `admin` |
| `INFLUXDB_PASSWORD` | Senha do admin (mínimo 8 caracteres) | **obrigatória** |
| `INFLUXDB_ORG` | Organização | `esp32` |
| `INFLUXDB_BUCKET` | Bucket dos dados | `sensors` |
| `INFLUXDB_RETENTION` | Janela de retenção (`0` = infinita) | `0` |
| `INFLUXDB_TOKEN` | Token de admin | **obrigatório** |

`INFLUXDB_PASSWORD` e `INFLUXDB_TOKEN` não têm valor padrão de propósito: o
compose falha na hora se estiverem vazios, em vez de subir um banco com
credencial previsível.

Gere o token com:

```bash
openssl rand -base64 48 | tr -d '\n='
```

O mesmo valor precisa ir em três lugares:

| Onde | Variável |
|---|---|
| aqui | `INFLUXDB_TOKEN` |
| [`services/subscriber/.env`](../subscriber/.env.example) | `INFLUX_TOKEN` |
| [`services/grafana/.env`](../grafana/.env.example) | `INFLUXDB_TOKEN` |

E `INFLUXDB_ORG` / `INFLUXDB_BUCKET` precisam bater com `INFLUX_ORG` /
`INFLUX_BUCKET` do subscriber.

> **O provisionamento só acontece uma vez.** `DOCKER_INFLUXDB_INIT_MODE=setup`
> age apenas quando o volume `influxdb_data` está vazio. Trocar a senha ou o
> token no `.env` depois disso não muda nada no banco já criado — é preciso
> alterar pela UI/CLI, ou apagar o volume (`docker compose down -v`, que
> **destrói o histórico**).

## Como rodar

```bash
cp .env.example .env      # defina INFLUXDB_PASSWORD e INFLUXDB_TOKEN
docker compose up -d
docker compose logs -f
```

Parar:

```bash
docker compose down          # mantém o histórico
docker compose down -v       # apaga os dados E o provisionamento
```

## Como validar

```bash
# Serviço no ar?
curl -s http://localhost:8086/health
# {"name":"influxdb","message":"ready for queries and writes","status":"pass",...}

# O healthcheck do container passou?
docker compose ps
# STATUS deve mostrar "Up (healthy)"
```

Confirme que o token e a organização funcionam:

```bash
docker compose exec influxdb influx bucket list \
  --org esp32 --token "$INFLUXDB_TOKEN"
```

Depois que o subscriber gravar algo, confira se os dados chegaram:

```bash
docker compose exec influxdb influx query --org esp32 --token "$INFLUXDB_TOKEN" '
from(bucket: "sensors")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "telemetry")
  |> last()
'
```

A UI fica em <http://localhost:8086> — entre com `INFLUXDB_USERNAME` e
`INFLUXDB_PASSWORD` e use o **Data Explorer** para montar consultas
visualmente.

## Consultas úteis (Flux)

Substituem o PromQL da arquitetura anterior. Prontas para colar no Grafana ou
no Data Explorer.

Temperatura da última hora:

```flux
from(bucket: "sensors")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "telemetry" and r._field == "temperature")
```

Média de 5 minutos (suaviza o gráfico):

```flux
from(bucket: "sensors")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "telemetry" and r._field == "temperature")
  |> aggregateWindow(every: 5m, fn: mean, createEmpty: false)
```

Último valor de cada sensor (para painel *Stat*):

```flux
from(bucket: "sensors")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "telemetry" and r._field == "pressure")
  |> group(columns: ["device_id", "device_name"])
  |> last()
```

Estado reportado pelo sensor (`1` = OK, `0` = ERROR):

```flux
from(bucket: "sensors")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "healthcheck" and r._field == "status")
  |> last()
```

Segundos desde o último contato de cada dispositivo — é assim que se detecta
queda, já que o firmware não registra LWT:

```flux
from(bucket: "sensors")
  |> range(start: -24h)
  |> filter(fn: (r) => r._measurement == "healthcheck")
  |> group(columns: ["device_id", "device_name"])
  |> last()
  |> map(fn: (r) => ({ r with silencio_s: int(v: uint(v: now()) - uint(v: r._time)) / 1000000000 }))
  |> keep(columns: ["device_id", "device_name", "silencio_s"])
```

Uptime em segundos (o payload traz milissegundos):

```flux
from(bucket: "sensors")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "healthcheck" and r._field == "uptime_ms")
  |> map(fn: (r) => ({ r with _value: r._value / 1000.0 }))
```

## Problemas comuns

| Sintoma | Causa provável |
|---|---|
| `defina INFLUXDB_TOKEN no .env` ao subir | Falta o `.env`, ou a variável está vazia |
| `unauthorized: unauthorized access` nas escritas | Token do subscriber diferente do daqui |
| `bucket not found` | `INFLUX_BUCKET` do subscriber diferente de `INFLUXDB_BUCKET` |
| Senha nova não funciona | O `setup` já rodou; troque pela UI ou recrie o volume |
| Pontos aparecendo em 1970 | `timestamp` do firmware sem NTP — ver [contrato](../../docs/mqtt-contract.md) |
| Sobe e reinicia em loop | `INFLUXDB_PASSWORD` com menos de 8 caracteres |
