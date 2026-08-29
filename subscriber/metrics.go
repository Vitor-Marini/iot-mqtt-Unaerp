package main

import "github.com/prometheus/client_golang/prometheus"

var (
	temperatureMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "esp32_temperature",
			Help: "Temperatura atual do ESP32.",
		},
		[]string{"sensor_id", "sensor_model"},
	)

	pressureMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "esp32_pressure",
			Help: "Pressão atual do ESP32.",
		},
		[]string{"sensor_id", "sensor_model"},
	)

	altitudeMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "esp32_altitude",
			Help: "Altitude atual do ESP32.",
		},
		[]string{"sensor_id", "sensor_model"},
	)

	rssiMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "esp32_rssi",
			Help: "RSSI atual do ESP32.",
		},
		[]string{"sensor_id", "sensor_model"},
	)

	freeHeapMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "esp32_free_heap",
			Help: "Heap livre atual do ESP32.",
		},
		[]string{"sensor_id", "sensor_model"},
	)

	uptimeMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "esp32_uptime_ms",
			Help: "Uptime atual do ESP32 em milissegundos.",
		},
		[]string{"sensor_id", "sensor_model"},
	)

	statusMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "esp32_status",
			Help: "Estado reportado pelo ESP32. 1 = OK, 0 = ERROR.",
		},
		[]string{"sensor_id", "sensor_model"},
	)
)

func RegisterMetrics() {

	prometheus.MustRegister(temperatureMetric)
	prometheus.MustRegister(pressureMetric)
	prometheus.MustRegister(altitudeMetric)

	prometheus.MustRegister(rssiMetric)
	prometheus.MustRegister(freeHeapMetric)
	prometheus.MustRegister(uptimeMetric)
	prometheus.MustRegister(statusMetric)
}