import paho.mqtt.client as mqtt

def on_connect(client, userdata, flags, reason_code, properties):
    print("Connected to MQTT broker")
    client.subscribe("/temperature-data")

def on_message(client, userdata, message):
    print(f"Topic: {message.topic}")
    print(f"Payload: {message.payload.decode()}")

def main():
    mqtt_client = mqtt.Client(
        callback_api_version=mqtt.CallbackAPIVersion.VERSION2
    )

    mqtt_client.on_connect = on_connect
    mqtt_client.on_message = on_message

    mqtt_client.connect(host="localhost", port=1883)
    mqtt_client.loop_forever()

if __name__ == "__main__":
    main()