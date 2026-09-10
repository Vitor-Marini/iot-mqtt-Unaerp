package main

import (
	"fmt"
	"subscriber/models"
	"time"
	"context"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

func ProcessTelemetry(telemetryChan <-chan models.Telemetry) {

	for telemetry := range telemetryChan {
		
		fmt.Println("========== TELEMETRY ==========")
		fmt.Println("Sensor ID:", telemetry.SensorID)
		fmt.Println("Sensor Model:", telemetry.SensorModel)
		fmt.Println("Temperature:", telemetry.Temperature)
		fmt.Println("Pressure:", telemetry.Pressure)
		fmt.Println("Altitude:", telemetry.Altitude)
		fmt.Println("Timestamp:", telemetry.Timestamp)
		fmt.Println("===============================")

		tags := map[string]string{
			"sensor_id":    telemetry.SensorID,
			"sensor_model": telemetry.SensorModel,
		}

		fields := map[string]interface{}{
			"temperature": telemetry.Temperature,
			"pressure":    telemetry.Pressure,
			"altitude":    telemetry.Altitude,
		}

		point := write.NewPoint(
			"telemetry",
			tags,
			fields,
			time.Unix(telemetry.Timestamp, 0),
		)


		if err := influxWriteAPI.WritePoint(
			context.Background(),
			point,
		); err != nil {
			fmt.Println("Erro ao escrever healthcheck no InfluxDB:", err)
		}
	
	}
}

func ProcessHealthcheck(healthcheckChan <-chan models.Healthcheck) {

	for healthcheck := range healthcheckChan {
		
		fmt.Println("========= HEALTHCHECK =========")
		fmt.Println("Sensor ID:", healthcheck.SensorID)
		fmt.Println("Sensor Model:", healthcheck.SensorModel)
		fmt.Println("Status:", healthcheck.Status)
		fmt.Println("RSSI:", healthcheck.RSSI)
		fmt.Println("Free Heap:", healthcheck.FreeHeap)
		fmt.Println("Uptime:", healthcheck.UptimeMs)
		fmt.Println("Timestamp:", healthcheck.Timestamp)
		fmt.Println("===============================")
		

		status := 0.0

		if healthcheck.Status == "OK" {
			status = 1.0
		}

		tags := map[string]string{
			"sensor_id":    healthcheck.SensorID,
			"sensor_model": healthcheck.SensorModel,
		}

		fields := map[string]interface{}{
			"status":    status,
			"rssi":      healthcheck.RSSI,
			"free_heap": healthcheck.FreeHeap,
			"uptime_ms": healthcheck.UptimeMs,
		}

		point := write.NewPoint(
			"healthcheck",
			tags,
			fields,
			time.Unix(healthcheck.Timestamp, 0),
		)

		if err := influxWriteAPI.WritePoint(
			context.Background(),
			point,
		); err != nil {
			fmt.Println("Erro ao escrever healthcheck no InfluxDB:", err)
		}
	}
}