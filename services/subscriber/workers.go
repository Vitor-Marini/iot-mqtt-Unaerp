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
		fmt.Println("Device ID:", telemetry.DeviceID)
		fmt.Println("Device Name:", telemetry.DeviceName)
		fmt.Println("Sensor Model:", telemetry.SensorModel)
		fmt.Println("Temperature:", telemetry.Temperature)
		fmt.Println("Pressure:", telemetry.Pressure)
		fmt.Println("Altitude:", telemetry.Altitude)
		fmt.Println("Timestamp:", telemetry.Timestamp)
		fmt.Println("===============================")

		tags := map[string]string{
			"device_id":    telemetry.DeviceID,
			"device_name":  telemetry.DeviceName,
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
			fmt.Println("Erro ao escrever telemetry no InfluxDB:", err)
		}

	}
}

func ProcessHealthcheck(healthcheckChan <-chan models.Healthcheck) {

	for healthcheck := range healthcheckChan {
		
		fmt.Println("========= HEALTHCHECK =========")
		fmt.Println("Device ID:", healthcheck.DeviceID)
		fmt.Println("Device Name:", healthcheck.DeviceName)
		fmt.Println("Sensor Model:", healthcheck.SensorModel)
		fmt.Println("Version:", healthcheck.Version)
		fmt.Println("IP:", healthcheck.IP)
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

		// "unknown" em vez de vazio: e o que faz o painel de distribuicao de
		// versoes mostrar quantas placas ainda nao receberam o OTA.
		version := healthcheck.Version
		if version == "" {
			version = "unknown"
		}

		tags := map[string]string{
			"device_id":    healthcheck.DeviceID,
			"device_name":  healthcheck.DeviceName,
			"sensor_model": healthcheck.SensorModel,
		}

		// version e ip sao FIELDS, nao tags. A regra: tag precisa ser estavel
		// durante a vida da serie; o que muda no tempo e field.
		//
		//   version -- muda a cada OTA. Como tag, cada rollout fraturaria as
		//              quatro series de healthcheck daquela placa.
		//   ip      -- muda a cada lease de DHCP, sem limite. Como tag, criaria
		//              serie nova a cada renovacao.
		fields := map[string]interface{}{
			"status":    status,
			"version":   version,
			"ip":        healthcheck.IP,
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