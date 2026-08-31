# MQTT Test Suite (Broker & Subscriber)

Simple testing environment to run a local Mosquitto MQTT broker and listen to real-time ESP32 payloads directly in your terminal.

---

## 🚀 How to Run

### Step 1: Start the Local Broker
In your first terminal, start the Mosquitto broker using Docker:
```bash
./test/start_broker.sh
```

### Step 2: Listen to Payload Output
In a second terminal, start the Python subscriber:
```bash
./test/venv/bin/python test/subscriber.py
```

---

## 📡 Expected Output
When your ESP32 node connects and publishes telemetry or health-checks, `subscriber.py` prints:

```text
[2026-08-29 19:40:00] TOPIC: devices/A1B2C3D4E5F6/telemetry
PAYLOAD:
{
  "sensor_id": "A1B2C3D4E5F6",
  "sensor_model": "BMP280",
  "temperature": 24.50,
  "pressure": 1013.25,
  "altitude": 540.20,
  "timestamp": 1787960400
}
------------------------------------------------------------
```
