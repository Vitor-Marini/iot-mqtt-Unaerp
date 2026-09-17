VERSION ?= 1.0.0

.PHONY: help build compile upload monitor upload-monitor clean

help: ## Lista os comandos disponiveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed -e 's/:.*## /\t/' | column -t -s "$$(printf '\t')"

build: compile ## Atalho para compilar

compile: ## Compila o firmware (ex: make compile VERSION=1.0.5)
	PLATFORMIO_BUILD_FLAGS="-D FIRMWARE_VERSION='\"$(VERSION)\"'" pio run
	@mkdir -p ../ota-server/storage
	@cp .pio/build/esp32dev/firmware.bin ../ota-server/storage/firmware-$(VERSION).bin
	@cp .pio/build/esp32dev/firmware.bin ../ota-server/storage/firmware.bin
	@echo ""
	@echo "=================================================="
	@echo "✅ Firmware v$(VERSION) compilado com sucesso!"
	@echo "📁 Binario copiado para: services/ota-server/storage/firmware.bin"
	@echo "=================================================="

upload: ## Grava o firmware na placa conectada via USB
	pio run -t upload

monitor: ## Abre o monitor serial a 115200 baud
	pio device monitor

upload-monitor: ## Grava via USB e ja abre o monitor serial
	pio run -t upload -t monitor

clean: ## Limpa os arquivos de build
	pio run -t clean
