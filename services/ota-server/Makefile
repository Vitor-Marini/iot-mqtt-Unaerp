.PHONY: help build run tidy fmt vet docker-build up down logs trigger-all trigger-node

help: ## Lista os alvos disponiveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed -e 's/:.*## /\t/' | column -t -s "$$(printf '\t')"

build: ## Compila o binario em bin/ota-server
	go build -o bin/ota-server .

run: ## Executa localmente (auto-detecta IP, ou force: make run HOST=192.168.10.130)
	go run . $(HOST)

tidy: ## Sincroniza go.mod/go.sum
	go mod tidy

fmt: ## Formata o codigo
	go fmt ./...

vet: ## Analise estatica
	go vet ./...

docker-build: ## Constroi a imagem Docker
	docker build -t iot-ota-server:local .

up: ## Sobe o servico isolado
	docker compose up -d --build

down: ## Derruba o servico
	docker compose down

logs: ## Acompanha os logs
	docker compose logs -f

trigger-all: ## Dispara atualizacao OTA para toda a frota (broadcast)
	./trigger_ota.sh --target all

trigger-node: ## Dispara atualizacao para no especifico: make trigger-node MAC=A1B2C3D4E5F6
	./trigger_ota.sh --target $(MAC)
