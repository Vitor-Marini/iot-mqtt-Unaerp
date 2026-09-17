# Atalhos do ambiente FULL LOCAL (docker-compose.yml da raiz).
# Para rodar um servico isolado, use o compose dentro de services/<nome>/.

.PHONY: help up down restart logs ps clean urls firmware firmware-upload mock monitor network

help: ## Lista os alvos disponiveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed -e 's/:.*## /\t/' | column -t -s "$$(printf '\t')"

up: ## Sobe mosquitto, influxdb, subscriber e grafana
	docker compose up -d --build

down: ## Derruba tudo (mantem os volumes)
	docker compose down

restart: ## Reinicia todos os servicos
	docker compose restart

logs: ## Acompanha os logs (use S=nome para um servico so)
	docker compose logs -f $(S)

ps: ## Estado dos containers
	docker compose ps

clean: ## Derruba tudo e APAGA os volumes (historico do InfluxDB incluido)
	docker compose down -v

urls: ## Mostra os enderecos dos servicos
	@echo "  Grafana     http://localhost:$${GRAFANA_PORT:-3000}"
	@echo "  InfluxDB    http://localhost:$${INFLUXDB_PORT:-8086}"
	@echo "  Broker MQTT tcp://localhost:$${MQTT_PORT:-1883}"
	@echo "  OTA Server  http://localhost:$${OTA_PORT:-8080}"

mock: ## Publica telemetria falsa no broker local (precisa de mosquitto_pub)
	cd services/subscriber && ./mock_esp32.sh

firmware: ## Compila o firmware e copia automaticamente para o ota-server (ex: make firmware VERSION=1.0.1)
monitor: ## Abre a janela do monitor de health check (roda no host)
	cd services/healthcheck-monitor && MQTT_HOST=localhost python monitor.py

network: ## Cria a rede externa opcional (ver docs/deployment.md)
	docker network create $${IOT_NETWORK:-iot-estacao} || true

firmware: ## Compila o firmware do ESP32
	cd services/esp32-firmware && pio run
	@mkdir -p services/ota-server/storage
	@cp services/esp32-firmware/.pio/build/esp32dev/firmware.bin services/ota-server/storage/firmware-$(if $(VERSION),$(VERSION),latest).bin
	@cp services/esp32-firmware/.pio/build/esp32dev/firmware.bin services/ota-server/storage/firmware.bin
	@echo "✅ Firmware compilado e copiado para services/ota-server/storage/"

firmware-upload: ## Grava o firmware via USB e abre o monitor serial
	cd services/esp32-firmware && pio run --target upload --target monitor

ota-trigger: ## Dispara atualizacao OTA via terminal: make ota-trigger (ou MAC=A1B2C3D4E5F6)
	@cd services/ota-server && ./trigger_ota.sh $(if $(MAC),--target $(MAC),--target all)
