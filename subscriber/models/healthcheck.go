package models

type Healthcheck struct {
	SensorID    string `json:"sensor_id"`
	SensorModel string `json:"sensor_model"`
	Status      string `json:"status"`
	RSSI        int    `json:"rssi"`
	FreeHeap    int64  `json:"free_heap"`
	UptimeMs    int64  `json:"uptime_ms"`
	Timestamp   int64  `json:"timestamp"`
}