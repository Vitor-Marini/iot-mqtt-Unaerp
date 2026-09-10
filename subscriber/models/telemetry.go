package models

type Telemetry struct {
	SensorID    string  `json:"sensor_id"`
	SensorModel string  `json:"sensor_model"`
	Temperature float64 `json:"temperature"`
	Pressure    float64 `json:"pressure"`
	Altitude    float64 `json:"altitude"`
	Timestamp   int64   `json:"timestamp"`
}