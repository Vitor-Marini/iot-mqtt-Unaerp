# Contrato MQTT

Este documento é a **fonte da verdade** do sistema. O firmware do ESP32 produz
exatamente o que está descrito aqui, e o subscriber Go consome exatamente isto.
Qualquer mudança neste arquivo exige mudança nos dois lados.

## Broker

| Item | Valor |
|---|---|
| Protocolo | MQTT 3.1.1 |
| Porta | `1883` (TCP, sem TLS) |
| Autenticação | anônima (ambiente de laboratório — ver `services/mosquitto/README.md`) |
| QoS usado | `1` (at least once) |
| Retain | `false` em `/telemetry`, `true` em `/health-check` |

## Tópicos

| Tópico | Direção | Publicador | Assinante |
|---|---|---|---|
| `/telemetry` | ESP32 → broker | `esp32-firmware` | `subscriber` |
| `/health-check` | ESP32 → broker | `esp32-firmware` | `subscriber` |

## `/telemetry`

Publicado a cada `PUBLISH_INTERVAL_MS` (padrão: 5000 ms) com a leitura do BMP280.

```json
{
  "sensor_id": "24:6F:28:AA:BB:CC",
  "sensor_model": "BMP280",
  "temperature": 24.83,
  "pressure": 1013.42,
  "altitude": 118.7,
  "timestamp": "2026-08-31T14:03:21Z"
}
```

| Campo | Tipo | Unidade | Obrigatório | Faixa aceita |
|---|---|---|---|---|
| `sensor_id` | string | — | sim | não vazio (MAC do ESP32) |
| `sensor_model` | string | — | sim | não vazio (`BMP280`) |
| `temperature` | number | °C | sim | `-100` a `150` |
| `pressure` | number | hPa | sim | `300` a `1100` |
| `altitude` | number | m | sim | `-500` a `10000` |
| `timestamp` | string \| number | UTC | sim | RFC 3339 (`2026-08-31T14:03:21Z`) **ou** unix seconds |

As faixas acima são validadas pelo subscriber. Mensagem fora da faixa é
descartada e contabilizada em `weather_messages_invalid_total`.

O campo `timestamp` aceita as duas formas porque o ESP32 publica RFC 3339 depois
de sincronizar com NTP, mas cai para unix seconds (uptime) se o NTP falhar.

## `/health-check`

Publicado com `retain: true` logo após cada conexão e depois a cada
`HEALTH_INTERVAL_MS` (padrão: 30000 ms).

```json
{
  "sensor_id": "24:6F:28:AA:BB:CC",
  "status": "online",
  "uptime_s": 3721,
  "rssi": -58
}
```

| Campo | Tipo | Unidade | Obrigatório | Valores |
|---|---|---|---|---|
| `sensor_id` | string | — | sim | não vazio |
| `status` | string | — | sim | `online` \| `offline` |
| `uptime_s` | number | s | não | `>= 0` |
| `rssi` | number | dBm | não | típico `-100` a `0` |

### Last Will and Testament

O ESP32 registra um LWT no broker. Se a conexão cair sem `DISCONNECT` limpo, o
broker publica automaticamente em `/health-check`:

```json
{ "sensor_id": "24:6F:28:AA:BB:CC", "status": "offline" }
```

Isso faz `weather_sensor_up` ir a `0` sem esperar o timeout de staleness.

## Detecção de sensor offline

O subscriber marca `weather_sensor_up = 0` quando:

1. chega um `/health-check` com `status != "online"` (inclusive o LWT); **ou**
2. passam mais de `SENSOR_STALE_AFTER` (padrão: 90s) sem nenhuma mensagem
   válida daquele `sensor_id`.
