# Atalhos do ambiente de DESENVOLVIMENTO (docker-compose.yml da raiz).
# Para rodar um serviço isolado, use o compose dentro de services/<nome>/.

.PHONY: help up down restart logs ps clean urls firmware firmware-upload subscriber-test

help: ## Lista os alvos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed -e 's/:.*## /\t/' | column -t -s "$$(printf '\t')"

up: ## Sobe broker, subscriber, prometheus e grafana
	docker compose up -d --build

down: ## Derruba tudo (mantém os volumes)
	docker compose down

restart: ## Reinicia todos os serviços
	docker compose restart

logs: ## Acompanha os logs (use S=nome para um serviço só)
	docker compose logs -f $(S)

ps: ## Estado dos containers
	docker compose ps

clean: ## Derruba tudo e APAGA os volumes (dados históricos incluídos)
	docker compose down -v

urls: ## Mostra os endereços dos serviços
	@echo "  Grafana     http://localhost:$${GRAFANA_PORT:-3000}"
	@echo "  Prometheus  http://localhost:$${PROMETHEUS_PORT:-9090}"
	@echo "  Metricas    http://localhost:$${SUBSCRIBER_PORT:-2112}/metrics"
	@echo "  Broker MQTT tcp://localhost:$${MQTT_PORT:-1883}"

firmware: ## Compila o firmware do ESP32
	cd services/esp32-firmware && pio run

firmware-upload: ## Grava o firmware e abre o monitor serial
	cd services/esp32-firmware && pio run --target upload --target monitor

subscriber-test: ## Roda os testes do subscriber
	cd services/subscriber && go test ./... -race -cover
