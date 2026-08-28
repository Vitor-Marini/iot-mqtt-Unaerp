import json
import datetime


def prepare_payload(
    mockup_temperature,
    sensor_name,
    sensor_type,
):
    """Prepares the payload for MQTT."""

    ct = datetime.datetime.now()
    ts = ct.timestamp()

    sensor_data = {
        "sensor_name": sensor_name,
        "sensor_type": sensor_type,
        "temperature": mockup_temperature,
        "timestamp": ts,
    }

    return json.dumps(sensor_data)


