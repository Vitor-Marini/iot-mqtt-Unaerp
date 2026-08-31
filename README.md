# Estação Meteorológica IoT

Monorepo da simulação de uma estação meteorológica IoT, desenvolvida para a
disciplina de **Hardware Configurável e IoT** (UNAERP).

Um **ESP32** com sensor **BMP280** lê temperatura, pressão e altitude e publica
via **MQTT**. Um **subscriber em Go** valida os dados e os expõe como métricas,
o **Prometheus** guarda a série temporal e o **Grafana** desenha os painéis.

```
┌──────────────┐   MQTT    ┌──────────────┐   MQTT    ┌──────────────┐
│ esp32-       │ ────────► │  mosquitto   │ ────────► │  subscriber  │
│ firmware     │  :1883    │   (broker)   │  :1883    │     (Go)     │
│ ESP32+BMP280 │           └──────────────┘           └──────┬───────┘
└──────────────┘                                             │ :2112/metrics
                                                             ▼ (scrape)
                            ┌──────────────┐   PromQL  ┌──────────────┐
                            │   grafana    │ ◄──────── │  prometheus  │
                            │    :3000     │   :9090   │    (TSDB)    │
                            └──────────────┘           └──────────────┘
```

## Serviços

Cada serviço é **autossuficiente** e foi desenhado para rodar em uma máquina
separada. Nenhum depende de outro estar no mesmo host: toda referência cruzada
é um endereço `host:porta` configurável.

| Serviço | Stack | Docker | Porta | O que faz |
|---|---|:---:|---|---|
| [`services/esp32-firmware`](services/esp32-firmware) | C++ / PlatformIO | — | — | Lê o BMP280 e publica em MQTT |
| [`services/mosquitto`](services/mosquitto) | Eclipse Mosquitto | sim | `1883` | Broker MQTT |
| [`services/subscriber`](services/subscriber) | Go | sim | `2112` | Assina MQTT e expõe `/metrics` |
| [`services/prometheus`](services/prometheus) | Prometheus | sim | `9090` | Banco de dados temporal |
| [`services/grafana`](services/grafana) | Grafana | sim | `3000` | Dashboards |

O firmware é o único sem Docker: ele roda no hardware.

## Estrutura

```
iot-mqtt-Unaerp/
├── README.md                  # este arquivo
├── docker-compose.yml         # DEV: sobe os 4 serviços Docker numa rede só
├── .env.example               # variáveis do ambiente de desenvolvimento
├── Makefile                   # atalhos: make up, make logs, make firmware
├── .editorconfig
│
├── docs/
│   ├── architecture.md        # diagrama, fluxo de dados e decisões de projeto
│   └── mqtt-contract.md       # tópicos e schema dos payloads
│
└── services/
    ├── esp32-firmware/        # platformio.ini, src/, include/config.h.example
    ├── mosquitto/             # config/mosquitto.conf, docker-compose.yml
    ├── subscriber/            # cmd/, internal/, Dockerfile, docker-compose.yml
    ├── prometheus/            # config/prometheus.yml, targets/, docker-compose.yml
    └── grafana/               # provisioning/, dashboards/, docker-compose.yml
```

### Convenções

Todo serviço segue o mesmo formato, para dar para navegar entre eles sem
reaprender nada:

- `README.md` — o que faz, interface, configuração, como rodar, como validar.
- `docker-compose.yml` — sobe **só aquele serviço**, para o deploy real.
- `.env.example` — toda a configuração, com valores padrão. Copie para `.env`.
- `config/` ou `provisioning/` — arquivos montados como somente-leitura.

Nenhum segredo é versionado: `.env` e `include/config.h` estão no `.gitignore`.

### O contrato é o acoplamento

O firmware é C++ e o subscriber é Go — os dois não compartilham tipos. O que os
mantém em acordo é [`docs/mqtt-contract.md`](docs/mqtt-contract.md), que define
os tópicos, o schema e as faixas válidas. **Mudou o contrato, mudam os dois
lados.**

## Execução

### Desenvolvimento: tudo numa máquina

O `docker-compose.yml` da raiz sobe os quatro serviços Docker numa rede única,
onde os nomes resolvem por DNS. Serve para desenvolver e demonstrar sem
precisar de quatro máquinas:

```bash
cp .env.example .env
make up
make urls
```

| Serviço | Endereço |
|---|---|
| Grafana | <http://localhost:3000> (`admin`/`admin`) |
| Prometheus | <http://localhost:9090> |
| Métricas | <http://localhost:2112/metrics> |
| Broker MQTT | `tcp://localhost:1883` |

Outros atalhos: `make help`. Para derrubar: `make down`, ou `make clean` para
apagar também os volumes.

O firmware continua rodando no ESP32 físico — aponte o `MQTT_HOST` dele para o
IP desta máquina.

### Produção: uma máquina por serviço

Suba nesta ordem, porque cada serviço depende do anterior estar no ar:

```bash
# 1. Máquina do broker
cd services/mosquitto && docker compose up -d

# 2. Máquina do subscriber
cd services/subscriber
cp .env.example .env          # ajuste MQTT_BROKER_URL para o IP do broker
docker compose up -d --build

# 3. Máquina do Prometheus
cd services/prometheus
cp .env.example .env
# aponte config/targets/subscriber.json para o IP do subscriber
docker compose up -d

# 4. Máquina do Grafana
cd services/grafana
cp .env.example .env          # ajuste PROMETHEUS_URL para o IP do Prometheus
docker compose up -d

# 5. ESP32
cd services/esp32-firmware
cp include/config.h.example include/config.h   # ajuste WiFi e MQTT_HOST
pio run --target upload
```

Cada README traz a tabela de variáveis e o passo de validação do seu serviço.

Portas que precisam estar liberadas no firewall:

| Origem | Destino | Porta |
|---|---|---|
| ESP32 | mosquitto | `1883` |
| subscriber | mosquitto | `1883` |
| prometheus | subscriber | `2112` |
| grafana | prometheus | `9090` |
| navegador | grafana | `3000` |

## Estado atual

A estrutura, os arquivos Docker e a configuração de todos os serviços estão
prontos. Os dois serviços com código próprio estão como esqueleto documentado:

| Serviço | Estado |
|---|---|
| `mosquitto`, `prometheus`, `grafana` | Funcionais — sobem e se conectam |
| `esp32-firmware` | Esqueleto — `platformio.ini` e `config.h.example` prontos, `src/main.cpp` a implementar |
| `subscriber` | Esqueleto — pacotes, `Dockerfile` e config prontos, lógica a implementar |

O README de cada um traz a seção de implementação, com a ordem sugerida e as
dependências previstas.

## Documentação

- [`docs/architecture.md`](docs/architecture.md) — diagrama, fluxo de uma
  leitura ponta a ponta, topologia de máquinas e as decisões de projeto com
  suas alternativas descartadas.
- [`docs/mqtt-contract.md`](docs/mqtt-contract.md) — tópicos, schema dos
  payloads, faixas válidas, LWT e regra de detecção de sensor offline.

## Requisitos

| Ferramenta | Versão | Para quê |
|---|---|---|
| Docker Engine | 20.10+ | mosquitto, subscriber, prometheus, grafana |
| Docker Compose | v2 | orquestração |
| Go | 1.21+ | desenvolver o subscriber |
| PlatformIO Core | 6+ | compilar e gravar o firmware |
