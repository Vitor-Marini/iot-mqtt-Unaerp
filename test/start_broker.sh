#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "[MQTT Test] Stopping old broker container if running..."
docker stop mosquitto >/dev/null 2>&1 || true
docker rm mosquitto >/dev/null 2>&1 || true

echo "[MQTT Test] Starting Mosquitto Broker Docker container on port 1883..."
docker run -d \
  --name mosquitto \
  -p 1883:1883 \
  -v "${SCRIPT_DIR}/mosquitto.conf:/mosquitto/config/mosquitto.conf" \
  eclipse-mosquitto:2

echo "[MQTT Test] Mosquitto broker is running on port 1883!"
