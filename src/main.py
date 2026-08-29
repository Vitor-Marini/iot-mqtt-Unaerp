from sensors.temperature_sensor import mockup_temperature
from payload.payload import prepare_payload
from configs.broker_config import mqtt_broker_configs
import paho.mqtt.client as mqtt
import time

def main():

    mqtt_client = mqtt.Client(callback_api_version=mqtt.CallbackAPIVersion.VERSION2)
    mqtt_client.connect(host=mqtt_broker_configs["HOST"],port=mqtt_broker_configs["PORT"])

    while True:
        temperature = mockup_temperature(2,2,-50,100)
        payload = prepare_payload(temperature,"sensor1","temperature")
        mqtt_client.publish(topic=mqtt_broker_configs["TOPIC"],payload=payload)

        time.sleep(0.5)

if __name__ == "__main__":
    main()
