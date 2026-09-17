#!/bin/bash
#
# Publica telemetria e health-check falsos no broker, para testar o pipeline
# sem hardware.
#
# Tudo e sobrescrevivel por variavel de ambiente, para dar para simular VARIAS
# placas ao mesmo tempo e montar o dashboard multi-dispositivo antes de gravar
# qualquer ESP32:
#
#   DEVICE_ID=A1B2C3D4E5F6 DEVICE_NAME=Estacao-Lab     ./mock_esp32.sh &
#   DEVICE_ID=B2C3D4E5F6A1 DEVICE_NAME=Estacao-Varanda ./mock_esp32.sh &
#
# LEGACY=1 publica no formato ANTIGO (chave `sensor_id`, sem device_name /
# version / ip). Serve de teste de regressao da transicao: com uma placa legada
# ao lado de duas novas, as tres precisam aparecer corretamente no InfluxDB,
# porque a identidade vem do topico e nao do payload.
#
#   LEGACY=1 DEVICE_ID=CCDDEEFF0011 ./mock_esp32.sh

export LC_ALL=C

DEVICE_ID="${DEVICE_ID:-A1B2C3D4E5F6}"
DEVICE_NAME="${DEVICE_NAME:-Mock-$DEVICE_ID}"
SENSOR_MODEL="${SENSOR_MODEL:-BMP280}"
VERSION="${VERSION:-1.1.0}"
IP="${IP:-192.168.0.99}"
MQTT_HOST="${MQTT_HOST:-localhost}"
MQTT_PORT="${MQTT_PORT:-1883}"
LEGACY="${LEGACY:-0}"

UPTIME=0

echo "======================================"
echo "         MOCK ESP32 INICIADO"
echo "======================================"
echo "Device ID:    $DEVICE_ID"
if [ "$LEGACY" = "1" ]; then
    echo "Formato:      LEGADO (sensor_id, sem device_name)"
else
    echo "Device Name:  $DEVICE_NAME"
    echo "Version:      $VERSION"
fi
echo "Sensor Model: $SENSOR_MODEL"
echo "MQTT:         $MQTT_HOST:$MQTT_PORT"
echo "======================================"

while true
do
    # =====================================
    # TIMESTAMP
    # =====================================

    TIMESTAMP=$(date +%s)

    # =====================================
    # GERA TEMPERATURA
    # Entre 10.00 e 40.00 °C
    # =====================================

    TEMPERATURE=$(awk 'BEGIN {
        srand()
        printf "%.2f", 10 + rand() * 30
    }')

    PRESSURE=1013.25
    ALTITUDE=540.20

    # =====================================
    # TELEMETRY
    # =====================================

    if [ "$LEGACY" = "1" ]; then
        TELEMETRY_JSON=$(printf \
            '{"sensor_id":"%s","sensor_model":"%s","temperature":%s,"pressure":%s,"altitude":%s,"timestamp":%s}' \
            "$DEVICE_ID" "$SENSOR_MODEL" "$TEMPERATURE" "$PRESSURE" "$ALTITUDE" "$TIMESTAMP"
        )
    else
        TELEMETRY_JSON=$(printf \
            '{"device_id":"%s","device_name":"%s","sensor_model":"%s","temperature":%s,"pressure":%s,"altitude":%s,"timestamp":%s}' \
            "$DEVICE_ID" "$DEVICE_NAME" "$SENSOR_MODEL" "$TEMPERATURE" "$PRESSURE" "$ALTITUDE" "$TIMESTAMP"
        )
    fi

    mosquitto_pub \
        -h "$MQTT_HOST" \
        -p "$MQTT_PORT" \
        -t "devices/$DEVICE_ID/telemetry" \
        -m "$TELEMETRY_JSON"

    echo "[TELEMETRY] temperature=${TEMPERATURE}°C pressure=${PRESSURE}hPa altitude=${ALTITUDE}m"

    # =====================================
    # HEALTHCHECK
    # A cada 5 segundos
    # =====================================

    if (( UPTIME % 5 == 0 )); then

        if [ "$LEGACY" = "1" ]; then
            HEALTHCHECK_JSON=$(printf \
                '{"sensor_id":"%s","sensor_model":"%s","status":"OK","rssi":-65,"free_heap":215400,"uptime_ms":%s,"timestamp":%s}' \
                "$DEVICE_ID" "$SENSOR_MODEL" "$((UPTIME * 1000))" "$TIMESTAMP"
            )
        else
            HEALTHCHECK_JSON=$(printf \
                '{"device_id":"%s","device_name":"%s","sensor_model":"%s","version":"%s","status":"OK","ip":"%s","rssi":-65,"free_heap":215400,"uptime_ms":%s,"timestamp":%s}' \
                "$DEVICE_ID" "$DEVICE_NAME" "$SENSOR_MODEL" "$VERSION" "$IP" "$((UPTIME * 1000))" "$TIMESTAMP"
            )
        fi

        mosquitto_pub \
            -h "$MQTT_HOST" \
            -p "$MQTT_PORT" \
            -t "devices/$DEVICE_ID/health-check" \
            -m "$HEALTHCHECK_JSON"

        echo "[HEALTHCHECK] status=OK rssi=-65 uptime=${UPTIME}s"
    fi

    # =====================================
    # INCREMENTA UPTIME
    # =====================================

    UPTIME=$((UPTIME + 1))

    # =====================================
    # AGUARDA 1 SEGUNDO
    # =====================================

    sleep 1
done
