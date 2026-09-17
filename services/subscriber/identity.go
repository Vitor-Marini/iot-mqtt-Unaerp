package main

import (
	"log"
	"regexp"
	"strings"
)

// Resolucao da identidade do dispositivo.
//
// O MAC vem do SEGMENTO DO TOPICO, nao do payload. O broker roteou a mensagem
// por esse caminho, entao ele existe sempre -- inclusive quando a placa ainda
// roda firmware antigo, que publica a chave `sensor_id` em vez de `device_id`.
// Isso e o que torna o rename do payload seguro com a frota em campo.
//
// As chaves do payload servem de conferencia: divergencia gera aviso no log,
// mas quem manda e o topico.

// resolveDeviceID devolve o MAC a ser usado como tag `device_id`.
func resolveDeviceID(fromTopic, payloadDeviceID, payloadSensorID, topic string) string {
	// Formato novo: confere e avisa se divergir.
	if payloadDeviceID != "" && payloadDeviceID != fromTopic {
		log.Printf(
			"[identidade] device_id do payload (%s) difere do topico (%s) em %s; usando o topico",
			payloadDeviceID, fromTopic, topic,
		)
	}

	// Formato antigo: mesma conferencia, sem tratar como erro.
	if payloadDeviceID == "" && payloadSensorID != "" && payloadSensorID != fromTopic {
		log.Printf(
			"[identidade] sensor_id legado (%s) difere do topico (%s) em %s; usando o topico",
			payloadSensorID, fromTopic, topic,
		)
	}

	return fromTopic
}

// resolveDeviceName cai para o proprio MAC quando o firmware ainda nao publica
// `device_name`, para a legenda do Grafana nunca ficar vazia.
func resolveDeviceName(payloadDeviceName, deviceID string) string {
	if payloadDeviceName != "" {
		return payloadDeviceName
	}
	return deviceID
}

// macRe valida o segmento do topico. O MAC vem de getDeviceMacId() no
// firmware: 12 hex maiusculos, sem separadores.
var macRe = regexp.MustCompile(`^[0-9A-F]{12}$`)

// deviceIDFromTopic extrai e valida o MAC de `devices/{MAC}/{sufixo}`.
//
// A validacao importa porque o topico virou carga critica: um publicador de
// teste em `devices/qualquer-coisa/telemetry` criaria uma serie permanente e
// poluiria o dropdown de dispositivos do Grafana para sempre.
func deviceIDFromTopic(topic string) (string, bool) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 {
		return "", false
	}

	id := strings.ToUpper(parts[1])
	if !macRe.MatchString(id) {
		return "", false
	}

	return id, true
}

// resolveSensorModel nunca devolve string vazia.
//
// O InfluxDB trata tag com valor vazio como AUSENTE, o que muda a chave da
// serie: um payload sem `sensor_model` faria aquele ponto cair numa serie
// diferente da dos demais, sem erro nenhum no log.
func resolveSensorModel(payloadSensorModel string) string {
	if payloadSensorModel != "" {
		return payloadSensorModel
	}
	return "unknown"
}
