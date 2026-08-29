#include "sensor_stubs.h"
#include "config.h"
#include <Wire.h>
#include <Adafruit_BMP280.h>

SensorManager sensorService;
static Adafruit_BMP280 bmp;

SensorManager::SensorManager() : initialized(false) {}

bool SensorManager::init() {
    Serial.printf("[Sensors] Initializing BMP280 on I2C (SDA: %d, SCL: %d)...\n", I2C_SDA, I2C_SCL);
    Wire.begin(I2C_SDA, I2C_SCL);

    initialized = bmp.begin(0x76);
    if (!initialized) {
        initialized = bmp.begin(0x77);
    }

    if (initialized) {
        uint8_t chipID = bmp.sensorID();
        Serial.printf("[Sensors] BMP280 connected successfully. Chip ID: 0x%02X\n", chipID);
        bmp.setSampling(Adafruit_BMP280::MODE_NORMAL,
                        Adafruit_BMP280::SAMPLING_X2,
                        Adafruit_BMP280::SAMPLING_X16,
                        Adafruit_BMP280::FILTER_X16,
                        Adafruit_BMP280::STANDBY_MS_500);
    } else {
        Serial.println("[Sensors Error] BMP280 not found on I2C bus.");
    }

    return initialized;
}

SensorPayload SensorManager::readData() {
    SensorPayload payload;
    payload.timestamp_ms = millis();

    if (initialized) {
        payload.temperature = bmp.readTemperature();
        payload.pressure    = bmp.readPressure() / 100.0F;
        payload.altitude    = bmp.readAltitude(1013.25);
        payload.isValid     = true;

        Serial.printf("[Sensors] Temp: %.2f C | Press: %.2f hPa | Alt: %.2f m\n", 
                      payload.temperature, 
                      payload.pressure, 
                      payload.altitude);
    } else {
        payload.temperature = 0.0f;
        payload.pressure    = 0.0f;
        payload.altitude    = 0.0f;
        payload.isValid     = false;

        Serial.println("[Sensors Error] Read failed: sensor not initialized.");
    }

    return payload;
}
