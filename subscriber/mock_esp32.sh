#!/bin/bash

SENSOR_ID="A1B2C3D4E5F6"
SENSOR_MODEL="BMP280"

TEMPERATURE=0.1
PRESSURE=1013.25
ALTITUDE=540.20

UPTIME=0

echo "======================================"
echo "      MOCK ESP32 INICIADO"
echo "======================================"
echo "Sensor ID: $SENSOR_ID"
echo "Sensor Model: $SENSOR_MODEL"
echo "MQTT: localhost:1883"
echo "======================================"

while true
do
    # Timestamp Unix atual
    TIMESTAMP=$(date +%s)

    # -----------------------------------
    # TELEMETRY
    # -----------------------------------

    mosquitto_pub \
        -h localhost \
        -p 1883 \
        -t "esp32/telemetry" \
        -m "{
            \"sensor_id\": \"$SENSOR_ID\",
            \"sensor_model\": \"$SENSOR_MODEL\",
            \"temperature\": $TEMPERATURE,
            \"pressure\": $PRESSURE,
            \"altitude\": $ALTITUDE,
            \"timestamp\": $TIMESTAMP
        }"

    echo "[TELEMETRY] temperature=$TEMPERATURE pressure=$PRESSURE altitude=$ALTITUDE"

    # -----------------------------------
    # HEALTHCHECK
    # -----------------------------------

    if (( UPTIME % 5 == 0 )); then

        mosquitto_pub \
            -h localhost \
            -p 1883 \
            -t "esp32/healthcheck" \
            -m "{
                \"sensor_id\": \"$SENSOR_ID\",
                \"sensor_model\": \"$SENSOR_MODEL\",
                \"status\": \"OK\",
                \"rssi\": -65,
                \"free_heap\": 215400,
                \"uptime_ms\": $((UPTIME * 1000)),
                \"timestamp\": $TIMESTAMP
            }"

        echo "[HEALTHCHECK] status=OK rssi=-65 uptime=${UPTIME}s"
    fi

    # Incrementa uptime
    UPTIME=$((UPTIME + 1))

    # Aguarda 1 segundo
    sleep 1
done

