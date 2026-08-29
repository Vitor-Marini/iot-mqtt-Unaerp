package main

import (
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"subscriber/models"
	"encoding/json"
)

func messageHandler(
	client mqtt.Client,
	msg mqtt.Message,
	telemetryChan chan<- models.Telemetry,
	healthcheckChan chan<- models.Healthcheck,
) {

	switch msg.Topic() {

	case "esp32/telemetry":

		var telemetry models.Telemetry

		if err := json.Unmarshal(msg.Payload(), &telemetry); err != nil {
			log.Println("Erro ao decodificar telemetry:", err)
			return
		}

		telemetryChan <- telemetry

	case "esp32/healthcheck":

		var healthcheck models.Healthcheck

		if err := json.Unmarshal(msg.Payload(), &healthcheck); err != nil {
			log.Println("Erro ao decodificar healthcheck:", err)
			return
		}

		healthcheckChan <- healthcheck

	default:

		log.Println("Tópico desconhecido:", msg.Topic())
	}
}