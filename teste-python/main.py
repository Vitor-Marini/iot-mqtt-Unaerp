from sensors.temperature_sensor import mockup_temperature
from payload.payload import prepare_payload
import paho.mqtt.client as mqtt
import time

def main():

    mqtt_client = mqtt.Client(callback_api_version=mqtt.CallbackAPIVersion.VERSION2)
    mqtt_client.connect(host="localhost",port=1883)

    while True:
        temperature = mockup_temperature(2,2,-50,100)
        payload = prepare_payload(temperature,"sensor1","temperature")
        mqtt_client.publish(topic="/temperature-data",payload=payload)

        time.sleep(0.5)

if __name__ == "__main__":
    main()
