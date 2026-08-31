.PHONY: help build run test lint tidy docker-build up down logs

help: ## Lista os alvos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Compila o binário em bin/subscriber
	go build -o bin/subscriber ./cmd/subscriber

run: ## Executa localmente (usa as variáveis do ambiente atual)
	go run ./cmd/subscriber

test: ## Roda os testes unitários
	go test ./... -race -cover

lint: ## Vet + verificação de formatação
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "arquivos mal formatados:"; gofmt -l .; exit 1)

tidy: ## Sincroniza go.mod/go.sum
	go mod tidy

docker-build: ## Constrói a imagem Docker
	docker build -t iot-subscriber:local .

up: ## Sobe o serviço isolado
	docker compose up -d --build

down: ## Derruba o serviço
	docker compose down

logs: ## Acompanha os logs
	docker compose logs -f
