package models

type Telemetry struct {
	// DeviceID e o MAC da placa. Preenchido pelo subscriber a partir do
	// segmento do topico, nao desta chave -- ver identity.go.
	DeviceID string `json:"device_id"`
	// SensorID e a chave antiga, publicada por firmware anterior ao rename.
	// Mantida so para conferencia durante a transicao da frota; remover quando
	// todas as placas estiverem atualizadas.
	SensorID    string  `json:"sensor_id"`
	DeviceName  string  `json:"device_name"`
	SensorModel string  `json:"sensor_model"`
	Temperature float64 `json:"temperature"`
	Pressure    float64 `json:"pressure"`
	Altitude    float64 `json:"altitude"`
	Timestamp   int64   `json:"timestamp"`
}
