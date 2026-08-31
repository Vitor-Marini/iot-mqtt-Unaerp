#ifndef SENSOR_STUBS_H
#define SENSOR_STUBS_H

#include <Arduino.h>

struct SensorPayload {
    float temperature;
    float pressure;
    float altitude;
    uint32_t timestamp_ms;
    bool isValid;
};

class SensorManager {
public:
    SensorManager();
    bool init();
    SensorPayload readData();

private:
    bool initialized;
};

extern SensorManager sensorService;

#endif // SENSOR_STUBS_H
