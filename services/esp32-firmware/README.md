# esp32-firmware

Firmware da estação meteorológica. Roda em um **ESP32** com sensor **BMP280**,
lê temperatura, pressão e altitude via I²C e publica em MQTT.

É o único serviço do monorepo sem Docker: ele roda no hardware.

## O que faz

- Conecta no WiFi e reconecta sozinho se a rede cair.
- Sincroniza o relógio por NTP para carimbar o payload em RFC 3339 UTC.
- Inicializa o BMP280 com o perfil *weather station* do datasheet
  (oversampling ×16 na pressão, filtro IIR ×16, standby de 500 ms).
- Publica em `/telemetry` a cada 5 s e em `/health-check` a cada 30 s.
- Registra um **LWT** (`status: "offline"`, retained) para que o subscriber
  detecte queda imediatamente, sem esperar o timeout de staleness.
- Reconecta ao broker com backoff exponencial de 1 s até 30 s.

O formato exato das mensagens está em [`docs/mqtt-contract.md`](../../docs/mqtt-contract.md).

## Hardware

| ESP32 | BMP280 |
|---|---|
| `3V3` | `VCC` |
| `GND` | `GND` |
| `GPIO 21` | `SDA` |
| `GPIO 22` | `SCL` |
| `GND` | `SDO` → endereço I²C `0x76` |

> Se o seu módulo vier com `SDO` em `VCC`, o endereço é `0x77`. Ajuste
> `BMP280_I2C_ADDRESS` no `config.h`.
>
> Cuidado: alguns módulos vendidos como BMP280 são na verdade **BME280**
> (que tem umidade). Se `bmp.begin()` falhar nos dois endereços, é provável que
> seja um BME280 — troque a lib para `Adafruit BME280 Library`.

## Pré-requisitos

- [PlatformIO Core](https://docs.platformio.org/en/latest/core/installation/index.html)
  (`pip install platformio`) ou a extensão PlatformIO no VS Code.
- Driver USB-serial da sua placa (CP2102 ou CH340).

## Configuração

As credenciais ficam em `include/config.h`, que **não é versionado**:

```bash
cp include/config.h.example include/config.h
```

Edite e preencha:

| Constante | Descrição | Padrão |
|---|---|---|
| `WIFI_SSID` / `WIFI_PASSWORD` | Rede WiFi 2.4 GHz (o ESP32 não fala 5 GHz) | — |
| `MQTT_HOST` / `MQTT_PORT` | Máquina que roda o serviço `mosquitto` | `—` / `1883` |
| `MQTT_USERNAME` / `MQTT_PASSWORD` | Vazios para conexão anônima | `""` |
| `BMP280_I2C_ADDRESS` | `0x76` ou `0x77` | `0x76` |
| `I2C_SDA_PIN` / `I2C_SCL_PIN` | Pinos I²C | `21` / `22` |
| `SEA_LEVEL_HPA` | Pressão ao nível do mar, base do cálculo de altitude | `1013.25` |
| `PUBLISH_INTERVAL_MS` | Intervalo de `/telemetry` | `5000` |
| `HEALTH_INTERVAL_MS` | Intervalo de `/health-check` | `30000` |
| `NTP_SERVER` | Servidor NTP | `pool.ntp.org` |

## Como rodar

```bash
# Compilar
pio run

# Gravar na placa (detecta a porta automaticamente)
pio run --target upload

# Monitor serial a 115200 baud
pio device monitor

# Gravar e abrir o monitor de uma vez
pio run --target upload --target monitor
```

Se você usa outra placa:

```bash
pio run -e esp32doit-devkit-v1 --target upload
```

Para forçar a porta:

```bash
pio run --target upload --upload-port COM5      # Windows
pio run --target upload --upload-port /dev/ttyUSB0   # Linux
```

## Como validar

No monitor serial você deve ver, em sequência:

```
=== Estacao Meteorologica ESP32 + BMP280 ===
[boot] sensor_id=24:6F:28:AA:BB:CC
[bmp280] ok
[wifi] conectando em minha-rede
[wifi] ok, ip=192.168.0.42 rssi=-58
[ntp] sincronizado
[mqtt] conectando em 192.168.0.10:1883 como esp32-24:6F:28:AA:BB:CC
[mqtt] conectado
[mqtt] {"sensor_id":"24:6F:28:AA:BB:CC","sensor_model":"BMP280","temperature":24.83,...}
```

Confirme do outro lado, na máquina do broker:

```bash
mosquitto_sub -h localhost -t '/telemetry' -v
```

## Problemas comuns

| Sintoma | Causa provável |
|---|---|
| `[bmp280] nao encontrado no endereco 0x76` | Fiação I²C invertida, ou o módulo usa `0x77` |
| `[wifi] falhou, reiniciando` | Rede 5 GHz, SSID/senha errados, ou sinal fraco |
| `[mqtt] falhou (rc=-2)` | `MQTT_HOST` inalcançável — firewall na porta 1883 do broker |
| `[ntp] indisponivel` | Sem saída para a internet; o firmware segue publicando com uptime |
| Altitude muito diferente da real | `SEA_LEVEL_HPA` desatualizado — é uma referência barométrica, não GPS |
