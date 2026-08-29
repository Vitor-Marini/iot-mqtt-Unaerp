#!/usr/bin/env env
import sys
import json
import time
import paho.mqtt.client as mqtt

BROKER_HOST = "localhost"
BROKER_PORT = 1883
TOPIC_PATTERN = "#"

def on_connect(client, userdata, flags, rc, properties=None):
    if rc == 0:
        print(f"[SUCCESS] Connected to MQTT Broker at {BROKER_HOST}:{BROKER_PORT}")
        print(f"[LISTENING] Subscribed to topic pattern: '{TOPIC_PATTERN}'\n" + "-" * 60)
        client.subscribe(TOPIC_PATTERN)
    else:
        print(f"[ERROR] Failed to connect, return code {rc}")

def on_message(client, userdata, msg):
    timestamp = time.strftime("%Y-%m-%d %H:%M:%S")
    topic = msg.topic
    raw_payload = msg.payload.decode("utf-8", errors="replace")

    print(f"[{timestamp}] TOPIC: {topic}")
    
    try:
        data = json.loads(raw_payload)
        pretty_json = json.dumps(data, indent=2)
        print(f"PAYLOAD:\n{pretty_json}")
    except json.JSONDecodeError:
        print(f"PAYLOAD: {raw_payload}")
    
    print("-" * 60)

def main():
    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)
    client.on_connect = on_connect
    client.on_message = on_message

    print(f"[MQTT Test] Connecting to broker {BROKER_HOST}:{BROKER_PORT}...")
    try:
        client.connect(BROKER_HOST, BROKER_PORT, 60)
        client.loop_forever()
    except KeyboardInterrupt:
        print("\n[STOP] Subscriber stopped by user.")
    except Exception as e:
        print(f"[ERROR] Connection error: {e}")

if __name__ == "__main__":
    main()
