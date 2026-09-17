package models

type Healthcheck struct {
	// DeviceID e o MAC da placa. Preenchido pelo subscriber a partir do
	// segmento do topico, nao desta chave -- ver identity.go.
	DeviceID string `json:"device_id"`
	// SensorID e a chave antiga, publicada por firmware anterior ao rename.
	SensorID    string `json:"sensor_id"`
	DeviceName  string `json:"device_name"`
	SensorModel string `json:"sensor_model"`
	// Version e IP vem do firmware desde a feature de OTA.
	Version   string `json:"version"`
	IP        string `json:"ip"`
	Status    string `json:"status"`
	RSSI      int    `json:"rssi"`
	FreeHeap  int64  `json:"free_heap"`
	UptimeMs  int64  `json:"uptime_ms"`
	Timestamp int64  `json:"timestamp"`
}
