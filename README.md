# Estação Meteorológica IoT

Monorepo da simulação de uma estação meteorológica IoT, desenvolvida para a
disciplina de **Hardware Configurável e IoT** (UNAERP).

Um **ESP32** com sensor **BMP280** lê temperatura, pressão e altitude e publica
via **MQTT**. Um **subscriber em Go** grava os dados no **InfluxDB**, e o
**Grafana** desenha os painéis.

```
┌──────────────┐   MQTT    ┌──────────────┐   MQTT    ┌──────────────┐
│ esp32-       │ ────────► │  mosquitto   │ ────────► │  subscriber  │
│ firmware     │  :1883    │   (broker)   │  :1883    │     (Go)     │
│ ESP32+BMP280 │           └──────────────┘           └──────┬───────┘
└──────────────┘                                             │ write
                                                             ▼ (line protocol)
                            ┌──────────────┐    Flux   ┌──────────────┐
                            │   grafana    │ ◄──────── │   influxdb   │
                            │    :3000     │   :8086   │    (TSDB)    │
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
| [`services/subscriber`](services/subscriber) | Go | sim | — | Assina MQTT e grava no InfluxDB |
| [`services/influxdb`](services/influxdb) | InfluxDB 2 | sim | `8086` | Banco de dados temporal |
| [`services/grafana`](services/grafana) | Grafana | sim | `3000` | Dashboards |

O firmware é o único sem Docker: ele roda no hardware. O subscriber é o único
sem porta: só faz conexões de saída.

## Estrutura

```
iot-mqtt-Unaerp/
├── README.md                  # este arquivo
├── docker-compose.yml         # FULL LOCAL: sobe os 4 serviços Docker numa rede só
├── .env.example               # variáveis do ambiente full local
├── Makefile                   # atalhos: make up, make logs, make mock
├── .editorconfig
│
├── docs/
│   ├── architecture.md        # diagrama, fluxo de dados e decisões de projeto
│   └── mqtt-contract.md       # tópicos e schema dos payloads
│
├── test/                      # bancada de teste: broker local + subscriber de inspeção
│
└── services/
    ├── esp32-firmware/        # platformio.ini, secrets.ini.example, include/, src/
    ├── mosquitto/             # config/mosquitto.conf, docker-compose.yml
    ├── subscriber/            # main.go, mqtt.go, workers.go, models/, Dockerfile
    ├── influxdb/              # docker-compose.yml, .env.example
    └── grafana/               # provisioning/, dashboards/, docker-compose.yml
```

### Convenções

Todo serviço segue o mesmo formato, para dar para navegar entre eles sem
reaprender nada:

- `README.md` — o que faz, interface, configuração, como rodar, como validar.
- `docker-compose.yml` — sobe **só aquele serviço**, para o deploy real.
- `.env.example` — toda a configuração, com valores padrão. Copie para `.env`.
- `config/` ou `provisioning/` — arquivos montados como somente-leitura.

Nenhum segredo é versionado: os `.env` e o `secrets.ini` do firmware estão no
`.gitignore`.

### O contrato é o acoplamento

O firmware é C++ e o subscriber é Go — os dois não compartilham tipos. O que os
mantém em acordo é [`docs/mqtt-contract.md`](docs/mqtt-contract.md), que define
os tópicos, o schema e as faixas válidas. **Mudou o contrato, mudam os dois
lados.**

### O token do InfluxDB aparece em três lugares

É a única configuração que precisa estar repetida e idêntica:

| Onde | Variável |
|---|---|
| [`services/influxdb/.env`](services/influxdb/.env.example) | `INFLUXDB_TOKEN` |
| [`services/subscriber/.env`](services/subscriber/.env.example) | **`TOKEN_INFLUX`** (nome invertido, é assim no código) |
| [`services/grafana/.env`](services/grafana/.env.example) | `INFLUXDB_TOKEN` |

No modo full local, o `.env` da raiz define `INFLUXDB_TOKEN` uma vez e o compose
distribui para os três.

## Execução

### Full local: tudo numa máquina

O `docker-compose.yml` da raiz sobe os quatro serviços Docker numa rede única,
onde os nomes resolvem por DNS. Serve para desenvolver e demonstrar sem
precisar de quatro máquinas:

```bash
cp .env.example .env
# defina INFLUXDB_PASSWORD e INFLUXDB_TOKEN — o compose falha sem eles
make up
make urls
```

| Serviço | Endereço |
|---|---|
| Grafana | <http://localhost:3000> (`admin`/`admin`) |
| InfluxDB | <http://localhost:8086> |
| Broker MQTT | `tcp://localhost:1883` |

Gere tráfego falso, sem precisar do ESP32:

```bash
make mock
```

Outros atalhos: `make help`. Para derrubar: `make down`, ou `make clean` para
apagar também os volumes.

O firmware continua rodando no ESP32 físico — aponte o `mqtt_host` do
`secrets.ini` dele para o IP desta máquina.

### Distribuído: uma máquina por serviço

Suba nesta ordem, porque cada serviço depende do anterior estar no ar:

```bash
# 1. Máquina do broker
cd services/mosquitto && docker compose up -d

# 2. Máquina do InfluxDB
cd services/influxdb
cp .env.example .env          # defina INFLUXDB_PASSWORD e INFLUXDB_TOKEN
docker compose up -d

# 3. Máquina do subscriber
cd services/subscriber
cp .env.example .env          # MQTT_HOST, INFLUX_HOST e TOKEN_INFLUX com IPs/token reais
docker compose up -d --build

# 4. Máquina do Grafana
cd services/grafana
cp .env.example .env          # INFLUXDB_URL e INFLUXDB_TOKEN
docker compose up -d

# 5. ESP32
cd services/esp32-firmware
cp secrets.ini.example secrets.ini   # ajuste WiFi, mqtt_host e mqtt_topic_base
pio run --target upload
```

Cada README traz a tabela de variáveis e o passo de validação do seu serviço.

Portas que precisam estar liberadas no firewall:

| Origem | Destino | Porta |
|---|---|---|
| ESP32 | mosquitto | `1883` |
| subscriber | mosquitto | `1883` |
| subscriber | influxdb | `8086` |
| grafana | influxdb | `8086` |
| navegador | grafana | `3000` |
| navegador | influxdb (UI, opcional) | `8086` |

Nada precisa alcançar o subscriber: ele não escuta em porta nenhuma.

## Estado atual

| Serviço | Estado |
|---|---|
| `esp32-firmware` | Completo — FreeRTOS, BMP280, WiFiManager, NTP e MQTT funcionando |
| `subscriber` | Completo — assina MQTT e grava no InfluxDB |
| `mosquitto`, `influxdb`, `grafana` | Funcionais — sobem e se conectam |
| Dashboards do Grafana | Vazios — a criar pela interface e exportar para `services/grafana/dashboards/` |

## Documentação

- [`docs/architecture.md`](docs/architecture.md) — diagrama, fluxo de uma
  leitura ponta a ponta, topologia de máquinas e as decisões de projeto com
  suas alternativas descartadas, incluindo por que saímos do Prometheus.
- [`docs/mqtt-contract.md`](docs/mqtt-contract.md) — tópicos, schema dos
  payloads, faixas válidas e regra de detecção de queda.
- [`services/influxdb/README.md`](services/influxdb/README.md) — consultas Flux
  prontas, que substituem o PromQL da arquitetura anterior.

## Requisitos

| Ferramenta | Versão | Para quê |
|---|---|---|
| Docker Engine | 20.10+ | mosquitto, subscriber, influxdb, grafana |
| Docker Compose | v2 | orquestração |
| Go | 1.26+ | desenvolver o subscriber |
| PlatformIO Core | 6+ | compilar e gravar o firmware |
| mosquitto-clients | qualquer | `make mock` e testes manuais |
