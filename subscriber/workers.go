package main

import (
	"subscriber/models"
)

func ProcessTelemetry(telemetryChan <-chan models.Telemetry) {

	for telemetry := range telemetryChan {
		/*
		fmt.Println("========== TELEMETRY ==========")
		fmt.Println("Sensor ID:", telemetry.SensorID)
		fmt.Println("Sensor Model:", telemetry.SensorModel)
		fmt.Println("Temperature:", telemetry.Temperature)
		fmt.Println("Pressure:", telemetry.Pressure)
		fmt.Println("Altitude:", telemetry.Altitude)
		fmt.Println("Timestamp:", telemetry.Timestamp)
		fmt.Println("===============================")
	*/

		temperatureMetric.WithLabelValues(
			telemetry.SensorID,
			telemetry.SensorModel,
		).Set(telemetry.Temperature)

		pressureMetric.WithLabelValues(
			telemetry.SensorID,
			telemetry.SensorModel,
		).Set(telemetry.Pressure)

		altitudeMetric.WithLabelValues(
			telemetry.SensorID,
			telemetry.SensorModel,
		).Set(telemetry.Altitude)
	}
}

func ProcessHealthcheck(healthcheckChan <-chan models.Healthcheck) {

	for healthcheck := range healthcheckChan {
		/*
		fmt.Println("========= HEALTHCHECK =========")
		fmt.Println("Sensor ID:", healthcheck.SensorID)
		fmt.Println("Sensor Model:", healthcheck.SensorModel)
		fmt.Println("Status:", healthcheck.Status)
		fmt.Println("RSSI:", healthcheck.RSSI)
		fmt.Println("Free Heap:", healthcheck.FreeHeap)
		fmt.Println("Uptime:", healthcheck.UptimeMs)
		fmt.Println("Timestamp:", healthcheck.Timestamp)
		fmt.Println("===============================")
		*/

		status := 0.0

		if healthcheck.Status == "OK" {
			status = 1.0
		}

		statusMetric.WithLabelValues(
			healthcheck.SensorID,
			healthcheck.SensorModel,
		).Set(status)

		rssiMetric.WithLabelValues(
			healthcheck.SensorID,
			healthcheck.SensorModel,
		).Set(float64(healthcheck.RSSI))

		freeHeapMetric.WithLabelValues(
			healthcheck.SensorID,
			healthcheck.SensorModel,
		).Set(float64(healthcheck.FreeHeap))

		uptimeMetric.WithLabelValues(
			healthcheck.SensorID,
			healthcheck.SensorModel,
		).Set(float64(healthcheck.UptimeMs))
	}
}