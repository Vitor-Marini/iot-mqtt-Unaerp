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

		parts := strings.Split(topic, "/")

		if len(parts) != 3 {
			log.Println("Tópico de telemetry inválido:", topic)
			return
		}

		deviceID := parts[1]

		var telemetry models.Telemetry

		if err := json.Unmarshal(msg.Payload(), &telemetry); err != nil {
			log.Println("Erro ao decodificar telemetry:", err)
			return
		}

		log.Printf(
			"Telemetry recebida do dispositivo %s",
			deviceID,
		)

		// Aqui você pode associar o deviceID à telemetry
		telemetryChan <- telemetry

	case strings.HasPrefix(topic, "devices/") &&
		strings.HasSuffix(topic, "/healthcheck"):

		parts := strings.Split(topic, "/")

		if len(parts) != 3 {
			log.Println("Tópico de telemetry inválido:", topic)
			return
		}

		deviceID := parts[1]

		var healthcheck models.Healthcheck

		if err := json.Unmarshal(msg.Payload(), &healthcheck); err != nil {
			log.Println("Erro ao decodificar healthcheck:", err)
			return
		}

		log.Printf(
			"Healthcheck recebida do dispositivo %s",
			deviceID,
		)

		healthcheckChan <- healthcheck

	default:

		log.Println("Tópico desconhecido:", topic)
	}
}