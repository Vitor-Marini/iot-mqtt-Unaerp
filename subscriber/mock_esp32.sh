#!/bin/bash


export LC_ALL=C

SENSOR_ID="A1B2C3D4E5F6"
SENSOR_MODEL="BMP280"

PRESSURE=1013.25
ALTITUDE=540.20

UPTIME=0

echo "======================================"
echo "         MOCK ESP32 INICIADO"
echo "======================================"
echo "Sensor ID:    $SENSOR_ID"
echo "Sensor Model: $SENSOR_MODEL"
echo "MQTT:         localhost:1883"
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

    # =====================================
    # TELEMETRY
    # =====================================

    TELEMETRY_JSON=$(printf \
        '{"sensor_id":"%s","sensor_model":"%s","temperature":%s,"pressure":%s,"altitude":%s,"timestamp":%s}' \
        "$SENSOR_ID" \
        "$SENSOR_MODEL" \
        "$TEMPERATURE" \
        "$PRESSURE" \
        "$ALTITUDE" \
        "$TIMESTAMP"
    )

    mosquitto_pub \
        -h localhost \
        -p 1883 \
        -t "devices/$SENSOR_ID/telemetry" \
        -m "$TELEMETRY_JSON"

    echo "[TELEMETRY] temperature=${TEMPERATURE}°C pressure=${PRESSURE}hPa altitude=${ALTITUDE}m"

    # =====================================
    # HEALTHCHECK
    # A cada 5 segundos
    # =====================================

    if (( UPTIME % 5 == 0 )); then

        HEALTHCHECK_JSON=$(printf \
            '{"sensor_id":"%s","sensor_model":"%s","status":"OK","rssi":-65,"free_heap":215400,"uptime_ms":%s,"timestamp":%s}' \
            "$SENSOR_ID" \
            "$SENSOR_MODEL" \
            "$((UPTIME * 1000))" \
            "$TIMESTAMP"
        )

        mosquitto_pub \
            -h localhost \
            -p 1883 \
            -t "devices/$SENSOR_ID/healthcheck" \
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