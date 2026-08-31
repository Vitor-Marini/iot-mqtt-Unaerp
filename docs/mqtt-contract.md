# Contrato MQTT

Este documento **descreve o firmware, não o contrário**. A fonte da verdade é o
código em [`services/esp32-firmware/`](../services/esp32-firmware/) — em
especial `src/mqtt_client.cpp` (montagem dos payloads) e `include/config.h`
(definição dos tópicos). Se o firmware mudar, este documento é que se ajusta.

Consumidores (subscriber Go, ferramentas de teste) devem seguir o que está aqui.

## Broker

| Item | Valor | Onde no código |
|---|---|---|
| Protocolo | MQTT 3.1.1 (PubSubClient) | `mqtt_client.cpp` |
| Host/porta | `MQTT_BROKER_HOST:MQTT_BROKER_PORT` | `secrets.ini` → build flags |
| Autenticação | `MQTT_USER` / `MQTT_PASS`, vazios = anônima | `secrets.ini` |
| Client ID | o MAC do dispositivo | `connect()` |
| QoS | `0` — PubSubClient só publica em QoS 0 | — |
| LWT | **não há** — `connect()` usa a forma de 3 argumentos | `mqtt_client.cpp` |

## Identidade do dispositivo (`sensor_id`)

Todo dispositivo se identifica pelo MAC de fábrica, formatado por
`getDeviceMacId()` em `include/config.h`:

```cpp
snprintf(macStr, sizeof(macStr), "%04X%08X", (uint16_t)(mac >> 32), (uint32_t)mac);
```

São **12 caracteres hexadecimais maiúsculos, sem separadores** — por exemplo
`A1B2C3D4E5F6`. Não é o formato `AA:BB:CC:DD:EE:FF` do `WiFi.macAddress()`.

O mesmo valor é usado como `sensor_id` no payload, como client ID no broker e
como segmento dos tópicos.

## Tópicos

Os tópicos são **hierárquicos e por dispositivo**, montados em
`SystemMQTTClient::init()`:

```cpp
telemetryTopic   = MQTT_TOPIC_BASE + "/" + macId + "/" + "telemetry";
healthCheckTopic = MQTT_TOPIC_BASE + "/" + macId + "/" + "health-check";
commandTopic     = MQTT_TOPIC_BASE + "/" + macId + "/" + "commands";
broadcastTopic   = MQTT_TOPIC_BASE + "/" + "broadcast";
```

`MQTT_TOPIC_BASE` vem de `mqtt_topic_base` no `secrets.ini` (padrão: `devices`).

| Tópico | Direção | Publicador | Assinante |
|---|---|---|---|
| `devices/{MAC}/telemetry` | ESP32 → broker | firmware | `subscriber` |
| `devices/{MAC}/health-check` | ESP32 → broker | firmware | `subscriber` |
| `devices/{MAC}/commands` | broker → ESP32 | (futuro: OTA) | firmware |
| `devices/broadcast` | broker → ESP32 | (futuro) | firmware |

Como o MAC entra no tópico, **quem consome precisa usar wildcard**:

```
devices/+/telemetry
devices/+/health-check
```

Esse desenho permite filtrar um nó específico sem varrer o fluxo inteiro e
suporta comando por difusão. O raciocínio completo está no
[README do firmware](../services/esp32-firmware/README.md#1-mqtt-topic-architecture).

## `devices/{MAC}/telemetry`

Publicado a cada `SENSOR_READ_INTERVAL_MS` (padrão 5000 ms), quando há leitura
na fila FreeRTOS e o MQTT está conectado.

```json
{
  "sensor_id": "A1B2C3D4E5F6",
  "sensor_model": "BMP280",
  "temperature": 24.5,
  "pressure": 1013.25,
  "altitude": 540.2,
  "timestamp": 1787960400
}
```

| Campo | Tipo | Unidade | Origem |
|---|---|---|---|
| `sensor_id` | string | — | `getDeviceMacId()` |
| `sensor_model` | string | — | literal `"BMP280"` |
| `temperature` | number | °C | `bmp.readTemperature()` |
| `pressure` | number | hPa | `bmp.readPressure() / 100.0` |
| `altitude` | number | m | `bmp.readAltitude(1013.25)` |
| `timestamp` | number | s | ver abaixo |

### `timestamp` é sempre um número

É **unix epoch em segundos**, nunca uma string. O firmware degrada em silêncio
quando o NTP não sincronizou:

```cpp
uint64_t timestampUtc = (now > 1000000000) ? (uint64_t)now
                                           : (uint64_t)(data.timestamp_ms / 1000);
```

Ou seja: com NTP, é a hora UTC real; sem NTP, é o **uptime da placa em
segundos** — um número pequeno, na casa das dezenas ou centenas. Quem consome
não deve confiar cegamente neste campo para ordenar séries; o instante de
recepção é mais confiável.

### Leitura inválida ainda é publicada

Se o BMP280 não inicializou, `SensorManager::readData()` devolve
`temperature`, `pressure` e `altitude` **zerados** com `isValid = false`, e a
telemetria é publicada assim mesmo. Pressão `0.0` hPa é fisicamente impossível
— é o sinal de sensor ausente. Cabe ao consumidor descartar essas leituras;
o `status` do health-check confirma (`"ERROR"`).

## `devices/{MAC}/health-check`

Publicado a cada `HEALTH_CHECK_INTERVAL_MS` (padrão 30000 ms) e também uma vez
logo após cada conexão bem-sucedida ao broker.

```json
{
  "sensor_id": "A1B2C3D4E5F6",
  "sensor_model": "BMP280",
  "status": "OK",
  "rssi": -58,
  "free_heap": 210376,
  "uptime_ms": 3721000,
  "timestamp": 1787960400
}
```

| Campo | Tipo | Unidade | Valores / origem |
|---|---|---|---|
| `sensor_id` | string | — | `getDeviceMacId()` |
| `sensor_model` | string | — | literal `"BMP280"` |
| `status` | string | — | **`"OK"` ou `"ERROR"`** |
| `rssi` | number | dBm | `WiFi.RSSI()` |
| `free_heap` | number | bytes | `ESP.getFreeHeap()` |
| `uptime_ms` | number | **ms** | `millis()` |
| `timestamp` | number | s | mesma regra da telemetria |

Dois detalhes que mudam a leitura do campo `status`:

- Os valores são `"OK"`/`"ERROR"`, **não** `"online"`/`"offline"`. Referem-se à
  saúde do **sensor**, não da conexão.
- O health-check disparado dentro de `connect()` passa `true` fixo, então o
  primeiro após cada reconexão relata `"OK"` mesmo com o BMP280 falhando. Só os
  periódicos refletem `isValid` de verdade.

`uptime_ms` está em **milissegundos** — divida por 1000 antes de comparar com
qualquer coisa em segundos.

## Detecção de dispositivo offline

**Não há LWT.** O firmware chama
`mqttClient.connect(clientId, MQTT_USER, MQTT_PASS)`, a forma de três
argumentos, que não registra Last Will. Se a placa cair, o broker não avisa
ninguém.

A única forma de detectar queda é por **ausência**: se não chegar mensagem de
um `sensor_id` por mais que `SENSOR_STALE_AFTER` (padrão 90 s, três vezes o
intervalo de health-check), o consumidor deve marcá-lo como offline. É assim
que o `subscriber` calcula `weather_sensor_up`.

## Observação: mensagens são retidas

O firmware publica com `mqttClient.publish(topic, buffer, n)`, onde `buffer` é
`char[256]` e `n` é o `size_t` devolvido por `serializeJson`.

Entre as sobrecargas do PubSubClient, `(const char*, const uint8_t*, unsigned
int)` não é viável — `char*` não converte implicitamente para `uint8_t*`. A
escolhida é `(const char*, const char*, boolean)`, e `n` (sempre > 0) vira
`retained = true`.

Consequência prática: **toda telemetria e todo health-check ficam retidos no
broker**. Quem assinar `devices/+/telemetry` recebe de imediato a última
mensagem de cada dispositivo, mesmo que seja antiga. O conteúdo do payload sai
correto, porque `serializeJson` termina o buffer em `\0`.

Isso aparenta ser não intencional (a intenção do `n` era o comprimento), mas o
firmware é a fonte da verdade e **não foi alterado**. Consumidores devem contar
com mensagens retidas na primeira assinatura.

## Referência cruzada

| Assunto | Onde |
|---|---|
| Arquitetura do firmware, fluxograma, decisões | [`services/esp32-firmware/README.md`](../services/esp32-firmware/README.md) |
| Comandos OTA em `devices/{MAC}/commands` | [`services/esp32-firmware/OTA_IMPLEMENTATION_GUIDE.md`](../services/esp32-firmware/OTA_IMPLEMENTATION_GUIDE.md) |
| Como o subscriber traduz isso em métricas | [`services/subscriber/README.md`](../services/subscriber/README.md) |
| Fluxo ponta a ponta | [`architecture.md`](architecture.md) |
