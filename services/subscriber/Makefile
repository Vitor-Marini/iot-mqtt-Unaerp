.PHONY: help build run tidy fmt vet docker-build up down logs mock

help: ## Lista os alvos disponiveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed -e 's/:.*## /\t/' | column -t -s "$$(printf '\t')"

build: ## Compila o binario em bin/subscriber
	go build -o bin/subscriber .

run: ## Executa localmente (le o .env do diretorio atual)
	go run .

tidy: ## Sincroniza go.mod/go.sum
	go mod tidy

fmt: ## Formata o codigo
	go fmt ./...

vet: ## Analise estatica
	go vet ./...

docker-build: ## Constroi a imagem Docker
	docker build -t iot-subscriber:local .

up: ## Sobe o servico isolado
	docker compose up -d --build

down: ## Derruba o servico
	docker compose down

logs: ## Acompanha os logs
	docker compose logs -f

mock: ## Publica telemetria falsa no broker local (precisa de mosquitto_pub)
	./mock_esp32.sh
