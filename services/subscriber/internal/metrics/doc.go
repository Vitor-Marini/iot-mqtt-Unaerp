// Package metrics registra e atualiza as métricas Prometheus da estação.
//
// Expõe os gauges weather_temperature_celsius, weather_pressure_hpa,
// weather_altitude_meters e weather_sensor_up (rotulados por sensor_id), além
// dos contadores weather_messages_received_total e
// weather_messages_invalid_total. Ver a tabela no README deste serviço.
package metrics
