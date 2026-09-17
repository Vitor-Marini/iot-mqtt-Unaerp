package main

import (
	"log"
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"subscriber/models"
	"encoding/json"
	"strings"
)

func messageHandler(
	client mqtt.Client,
	msg mqtt.Message,
	telemetryChan chan<- models.Telemetry,
	healthcheckChan chan<- models.Healthcheck,
) {


	fmt.Println("TOPICO:", msg.Topic())
	fmt.Println("PAYLOAD:", string(msg.Payload()))

	topic := msg.Topic()

	switch {
	case strings.HasPrefix(topic, "devices/") &&
		strings.HasSuffix(topic, "/telemetry"):

		deviceID, ok := deviceIDFromTopic(topic)
		if !ok {
			log.Println("Tópico de telemetry inválido:", topic)
			return
		}

		var telemetry models.Telemetry

		if err := json.Unmarshal(msg.Payload(), &telemetry); err != nil {
			log.Println("Erro ao decodificar telemetry:", err)
			return
		}

		// A identidade vem do topico, nao do payload (ver identity.go).
		telemetry.DeviceID = resolveDeviceID(
			deviceID, telemetry.DeviceID, telemetry.SensorID, topic,
		)
		telemetry.DeviceName = resolveDeviceName(telemetry.DeviceName, telemetry.DeviceID)
		telemetry.SensorModel = resolveSensorModel(telemetry.SensorModel)

		log.Printf(
			"Telemetry recebida do dispositivo %s (%s)",
			telemetry.DeviceID, telemetry.DeviceName,
		)

		telemetryChan <- telemetry

	case strings.HasPrefix(topic, "devices/") &&
		strings.HasSuffix(topic, "/health-check"):

		deviceID, ok := deviceIDFromTopic(topic)
		if !ok {
			log.Println("Tópico de health-check inválido:", topic)
			return
		}

		var healthcheck models.Healthcheck

		if err := json.Unmarshal(msg.Payload(), &healthcheck); err != nil {
			log.Println("Erro ao decodificar healthcheck:", err)
			return
		}

		healthcheck.DeviceID = resolveDeviceID(
			deviceID, healthcheck.DeviceID, healthcheck.SensorID, topic,
		)
		healthcheck.DeviceName = resolveDeviceName(healthcheck.DeviceName, healthcheck.DeviceID)
		healthcheck.SensorModel = resolveSensorModel(healthcheck.SensorModel)

		log.Printf(
			"Healthcheck recebida do dispositivo %s (%s)",
			healthcheck.DeviceID, healthcheck.DeviceName,
		)

		healthcheckChan <- healthcheck

	default:

		log.Println("Tópico desconhecido:", topic)
	}
}