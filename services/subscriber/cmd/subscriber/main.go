// Command subscriber assina os tópicos MQTT da estação meteorológica e expõe
// as leituras como métricas Prometheus.
//
// O que precisa ser implementado está descrito no README deste serviço:
// carregar a configuração, conectar no broker, assinar /telemetry e
// /health-check, alimentar o coletor de métricas e servir /metrics.
package main

func main() {
	// TODO: config.Load, conectar no broker, assinar os tópicos,
	// subir o servidor HTTP de métricas e tratar SIGINT/SIGTERM.
}
