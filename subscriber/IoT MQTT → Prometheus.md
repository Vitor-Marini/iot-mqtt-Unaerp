# IoT MQTT → Prometheus

Projeto de monitoramento de sensores ESP32 utilizando **MQTT**, **Mosquitto**, **Go** e **Prometheus**, com visualização através do **Grafana**.

A arquitetura foi projetada para funcionar inicialmente em uma rede local, mas permitindo alterar facilmente o endereço do broker MQTT quando os componentes forem distribuídos entre diferentes máquinas.

---

# 1. Arquitetura

O fluxo principal dos dados é:

```text
┌──────────────┐
│    ESP32     │
│              │
│ Telemetry    │
│ Healthcheck  │
└──────┬───────┘
       │
       │ MQTT
       ▼
┌──────────────┐
│  Mosquitto   │
│ MQTT Broker  │
└──────┬───────┘
       │
       │ MQTT
       ▼
┌────────────────────────┐
│     Subscriber Go      │
│                        │
│  MQTT Client (Paho)    │
└───────────┬────────────┘
            │
       ┌────┴─────┐
       │          │
       ▼          ▼
 telemetry    healthcheck
  Channel       Channel
       │          │
       ▼          ▼
 ProcessTelemetry  ProcessHealthcheck
       │          │
       └────┬─────┘
            │
            ▼
     Prometheus Metrics
            │
            ▼
        HTTP /metrics
            │
            ▼
       ┌────────────┐
       │ Prometheus │
       └─────┬──────┘
             │
             ▼
         ┌────────┐
         │ Grafana│
         └────────┘
```

---

# 2. Tecnologias

- **ESP32** — coleta e envia os dados dos sensores.
- **MQTT** — protocolo de comunicação.
- **Mosquitto** — broker MQTT.
- **Go** — subscriber e processamento das mensagens.
- **Paho MQTT** — biblioteca MQTT para Go.
- **Prometheus** — coleta e armazena as métricas.
- **Grafana** — visualização das métricas.

---

# 3. Tópicos MQTT

Atualmente existem dois tópicos:

```text
esp32/telemetry
esp32/healthcheck
```

## 3.1 Telemetry

Utilizado para informações relacionadas às medições do sensor.

Exemplo:

```json
{
  "sensor_id": "A1B2C3D4E5F6",
  "sensor_model": "BMP280",
  "temperature": 25.40,
  "pressure": 1013.25,
  "altitude": 540.20,
  "timestamp": 1756496700
}
```

---

## 3.2 Healthcheck

Utilizado para informações relacionadas ao estado do dispositivo.

Exemplo:

```json
{
  "sensor_id": "A1B2C3D4E5F6",
  "sensor_model": "BMP280",
  "status": "OK",
  "rssi": -65,
  "free_heap": 215400,
  "uptime_ms": 45000,
  "timestamp": 1756496700
}
```

O significado exato dos campos do payload é definido pelo firmware do ESP32.

O subscriber não deve alterar o payload recebido. Ele apenas decodifica e transforma os dados necessários em métricas.

---

# 4. Configuração do MQTT

O endereço do broker é definido através do `.env`:

```env
MQTT_HOST=localhost
MQTT_PORT=1883
SUBSCRIBER_ID=subscriber
```

Durante o desenvolvimento local:

```env
MQTT_HOST=localhost
```

Quando o Mosquitto estiver em outro computador da rede:

```env
MQTT_HOST=192.168.0.100
```

O código não precisa ser alterado.

A construção do endereço é feita através de:

```go
host := getEnv("MQTT_HOST", "localhost")
port := getEnv("MQTT_PORT", "1883")

broker := fmt.Sprintf("tcp://%s:%s", host, port)
```

---

# 5. Channels

O subscriber separa os tipos de mensagens utilizando channels:

```go
telemetryChan := make(chan models.Telemetry)
healthcheckChan := make(chan models.Healthcheck)
```

O fluxo é:

```text
                    MQTT
                     │
                     ▼
              messageHandler
                 /       \
                /         \
               ▼           ▼
       telemetryChan   healthcheckChan
             │                │
             ▼                ▼
    ProcessTelemetry   ProcessHealthcheck
```

Isso permite que cada tipo de mensagem seja processado separadamente.

---

# 6. Goroutines

Os processadores são executados em goroutines:

```go
go ProcessTelemetry(telemetryChan)
go ProcessHealthcheck(healthcheckChan)
```

Cada worker permanece aguardando novas mensagens:

```go
for telemetry := range telemetryChan {
    // processamento
}
```

e:

```go
for healthcheck := range healthcheckChan {
    // processamento
}
```

Dessa maneira, o recebimento das mensagens MQTT fica desacoplado do processamento das métricas.

---

# 7. Processamento da Telemetry

Quando uma mensagem chega em:

```text
esp32/telemetry
```

o `messageHandler` decodifica o JSON:

```go
json.Unmarshal(msg.Payload(), &telemetry)
```

Depois envia o objeto para:

```go
telemetryChan <- telemetry
```

O `ProcessTelemetry` recebe a informação e atualiza as métricas:

```text
Telemetry
    │
    ├── temperature → esp32_temperature
    │
    ├── pressure    → esp32_pressure
    │
    └── altitude    → esp32_altitude
```

---

# 8. Processamento do Healthcheck

Quando uma mensagem chega em:

```text
esp32/healthcheck
```

o JSON é convertido para:

```go
models.Healthcheck
```

e enviado para:

```go
healthcheckChan
```

O `ProcessHealthcheck` atualiza:

```text
Healthcheck
    │
    ├── rssi       → esp32_rssi
    │
    ├── free_heap  → esp32_free_heap
    │
    ├── uptime_ms  → esp32_uptime_ms
    │
    └── status     → esp32_status
```

---

# 9. Métricas Prometheus

As métricas são criadas utilizando `GaugeVec`.

Exemplo:

```go
temperatureMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "esp32_temperature",
		Help: "Temperatura atual do ESP32.",
	},
	[]string{"sensor_id", "sensor_model"},
)
```

As labels permitem separar os sensores.

Por exemplo:

```text
esp32_temperature{
    sensor_id="A1B2C3D4E5F6",
    sensor_model="BMP280"
} 25.4
```

Outro ESP32 poderá gerar:

```text
esp32_temperature{
    sensor_id="B2C3D4E5F6A1",
    sensor_model="BMP280"
} 28.1
```

Essas serão séries temporais diferentes.

---

# 10. Métricas atuais

Atualmente o subscriber disponibiliza:

| Métrica | Origem | Tipo |
|---|---|---|
| `esp32_temperature` | Telemetry | Gauge |
| `esp32_pressure` | Telemetry | Gauge |
| `esp32_altitude` | Telemetry | Gauge |
| `esp32_rssi` | Healthcheck | Gauge |
| `esp32_free_heap` | Healthcheck | Gauge |
| `esp32_uptime_ms` | Healthcheck | Gauge |
| `esp32_status` | Healthcheck | Gauge |

---

# 11. Métrica de Status

O campo:

```json
"status": "OK"
```

não pode ser exposto diretamente como uma métrica textual.

Por isso o subscriber converte o status para um valor numérico.

A regra atual é:

```text
┌───────────────┬──────────────┐
│ Status ESP32  │ esp32_status │
├───────────────┼──────────────┤
│ OK            │      1       │
│ ERROR         │      0       │
└───────────────┴──────────────┘
```

Exemplo:

```json
{
  "status": "OK"
}
```

resulta em:

```text
esp32_status{sensor_id="A1B2C3D4E5F6",sensor_model="BMP280"} 1
```

Enquanto:

```json
{
  "status": "ERROR"
}
```

resulta em:

```text
esp32_status{sensor_id="A1B2C3D4E5F6",sensor_model="BMP280"} 0
```

### Regra importante

**Somente `OK` é considerado saudável.**

Qualquer status diferente de `OK` será tratado como `0`.

Por exemplo:

```text
OK       → 1
ERROR    → 0
WARNING  → 0
UNKNOWN  → 0
```

Essa regra pode ser alterada futuramente caso o contrato do ESP32 defina outros estados.

---

# 12. IMPORTANTE: `esp32_status` não significa ONLINE/OFFLINE

A métrica:

```text
esp32_status
```

representa **somente o estado reportado pelo próprio ESP32**.

Ela não representa diretamente se o dispositivo está conectado ao MQTT.

Essas duas situações são diferentes.

---

## 12.1 ESP32 reportando ERROR

Imagine:

```text
ESP32
 │
 ├── healthcheck → OK
 │
 ├── healthcheck → ERROR
 │
 ├── healthcheck → ERROR
 │
 └── healthcheck → ERROR
```

Nesse caso:

```text
esp32_status = 0
```

Porém, o ESP32 continua enviando mensagens.

Portanto:

```text
Status reportado: ERROR
Comunicação:      funcionando
```

---

## 12.2 ESP32 parou de enviar

Agora imagine:

```text
ESP32
 │
 ├── healthcheck → OK
 │
 ├── healthcheck → OK
 │
 └── silêncio
       │
       │
       └── nenhuma mensagem
```

Nesse caso não recebemos um novo:

```text
status = ERROR
```

Simplesmente não recebemos mais nada.

Portanto:

```text
esp32_status
```

não deve ser utilizado sozinho para determinar se o ESP32 está offline.

Futuramente deverá existir uma métrica separada para representar a disponibilidade/comunicação do dispositivo.

---

# 13. Resumo da diferença

```text
                 ┌─────────────────────┐
                 │      ESP32          │
                 └──────────┬──────────┘
                            │
                     Healthcheck
                            │
              ┌─────────────┴─────────────┐
              │                           │
              ▼                           ▼
       status = ERROR              nenhuma mensagem
              │                           │
              ▼                           ▼
     esp32_status = 0             sem atualização
              │                           │
              ▼                           ▼
     ESP32 reportou erro           comunicação?
                                      ↓
                                será tratado
                              separadamente
```

Portanto:

```text
esp32_status = 0
```

significa:

> O ESP32 informou que seu estado atual é `ERROR`.

Não significa:

> O ESP32 está offline.

---

# 14. Endpoint `/metrics`

A aplicação Go disponibiliza as métricas através de HTTP:

```text
/metrics
```

Utilizando:

```go
http.Handle(
	"/metrics",
	promhttp.Handler(),
)
```

O servidor pode ficar disponível em:

```text
http://localhost:8080/metrics
```

O `promhttp.Handler()` consulta o registry padrão do Prometheus e transforma as métricas registradas em uma resposta HTTP no formato esperado pelo Prometheus.

---

# 15. Exemplo do `/metrics`

Depois que o ESP32 enviar seus dados, podemos encontrar:

```text
# HELP esp32_temperature Temperatura atual do ESP32.
# TYPE esp32_temperature gauge
esp32_temperature{sensor_id="A1B2C3D4E5F6",sensor_model="BMP280"} 25.4

# HELP esp32_pressure Pressão atual do ESP32.
# TYPE esp32_pressure gauge
esp32_pressure{sensor_id="A1B2C3D4E5F6",sensor_model="BMP280"} 1013.25

# HELP esp32_altitude Altitude atual do ESP32.
# TYPE esp32_altitude gauge
esp32_altitude{sensor_id="A1B2C3D4E5F6",sensor_model="BMP280"} 540.2

# HELP esp32_rssi RSSI atual do ESP32.
# TYPE esp32_rssi gauge
esp32_rssi{sensor_id="A1B2C3D4E5F6",sensor_model="BMP280"} -65

# HELP esp32_free_heap Heap livre atual do ESP32.
# TYPE esp32_free_heap gauge
esp32_free_heap{sensor_id="A1B2C3D4E5F6",sensor_model="BMP280"} 215400

# HELP esp32_uptime_ms Uptime atual do ESP32 em milissegundos.
# TYPE esp32_uptime_ms gauge
esp32_uptime_ms{sensor_id="A1B2C3D4E5F6",sensor_model="BMP280"} 45000

# HELP esp32_status Estado reportado pelo ESP32. 1 = OK, 0 = ERROR.
# TYPE esp32_status gauge
esp32_status{sensor_id="A1B2C3D4E5F6",sensor_model="BMP280"} 1
```

---

# 16. Valor atual vs histórico

O subscriber mantém o valor atual das métricas.

Por exemplo:

```text
esp32_temperature = 25.4
```

Depois:

```text
esp32_temperature = 25.8
```

Depois:

```text
esp32_temperature = 26.1
```

O subscriber não precisa armazenar manualmente:

```text
25.4
25.8
26.1
```

O Prometheus fará o scraping do endpoint `/metrics` e armazenará as amostras.

Assim, posteriormente será possível visualizar:

### Valor atual

```text
TEMPERATURA ATUAL

25.4 °C
```

### Histórico

```text
26 ┤              ╭──╮
25 ┤────╮───────╯  ╰──
24 ┤    ╰──╮
   └──────────────────
       tempo
```

A mesma métrica pode ser utilizada para as duas visualizações.

---

# 17. Métricas do próprio Go

Além das métricas do ESP32, o endpoint `/metrics` também apresenta métricas automáticas do runtime Go.

Exemplos:

```text
go_goroutines
go_memstats_alloc_bytes
go_threads
process_cpu_seconds_total
process_resident_memory_bytes
```

Essas métricas permitem futuramente monitorar o próprio subscriber.

Portanto, teremos dois níveis de monitoramento:

```text
┌──────────────────────────┐
│       ESP32              │
│                          │
│ temperatura              │
│ pressão                  │
│ altitude                 │
│ RSSI                     │
│ heap                     │
│ uptime                   │
│ status                   │
└──────────────────────────┘

┌──────────────────────────┐
│    Subscriber Go         │
│                          │
│ goroutines               │
│ memória                  │
│ CPU                      │
│ threads                  │
└──────────────────────────┘
```

---

# 18. Responsabilidade de cada componente

## ESP32

Responsável por:

```text
Coletar dados
     ↓
Montar payload
     ↓
Publicar MQTT
```

---

## Mosquitto

Responsável por:

```text
Receber mensagens MQTT
        ↓
Distribuir para subscribers
```

O Mosquitto não precisa conhecer o conteúdo dos payloads.

---

## Subscriber Go

Responsável por:

```text
Receber MQTT
     ↓
Decodificar JSON
     ↓
Processar dados
     ↓
Atualizar métricas
```

---

## Prometheus

Responsável por:

```text
Consultar /metrics
       ↓
Armazenar séries temporais
       ↓
Permitir consultas PromQL
```

---

## Grafana

Responsável por:

```text
Consultar Prometheus
       ↓
Transformar dados
       ↓
Criar dashboards
       ↓
Exibir gráficos
       ↓
Exibir status
       ↓
Criar alertas
```

---

# 19. Fluxo completo

Uma mensagem de temperatura percorre o sistema desta maneira:

```text
ESP32
 │
 │ MQTT
 ▼
Mosquitto
 │
 ▼
messageHandler()
 │
 │ json.Unmarshal()
 ▼
models.Telemetry
 │
 ▼
telemetryChan
 │
 ▼
ProcessTelemetry()
 │
 │ .Set(25.4)
 ▼
esp32_temperature
 │
 ▼
/metrics
 │
 │ HTTP
 ▼
Prometheus
 │
 ├── armazena histórico
 │
 └── executa consultas
          │
          ▼
       Grafana
```

Para o status:

```text
ESP32
 │
 │
 │ "status": "OK"
 ▼
Mosquitto
 │
 ▼
messageHandler()
 │
 ▼
Healthcheck
 │
 ▼
healthcheckChan
 │
 ▼
ProcessHealthcheck()
 │
 │ OK → 1
 ▼
esp32_status = 1
 │
 ▼
/metrics
 │
 ▼
Prometheus
```

Se o ESP32 enviar:

```json
{
  "status": "ERROR"
}
```

o fluxo será:

```text
"ERROR"
   │
   ▼
esp32_status = 0
```

---

# 20. Estrutura do projeto

Uma possível organização:

```text
subscriber/
│
├── main.go
├── mqtt.go
├── workers.go
├── metrics.go
├── server.go
│
├── models/
│   ├── telemetry.go
│   └── healthcheck.go
│
├── .env
├── go.mod
├── go.sum
└── README.md
```

Responsabilidades:

```text
main.go
    Inicialização da aplicação

mqtt.go
    Comunicação com Mosquitto

workers.go
    Processamento das mensagens

metrics.go
    Definição e registro das métricas

server.go
    Endpoint HTTP /metrics

models/
    Estrutura dos payloads recebidos
```

---

# 21. Objetivo final

A arquitetura final desejada é:

```text
             ┌──────────────┐
             │    ESP32     │
             └──────┬───────┘
                    │
                    │ MQTT
                    ▼
             ┌──────────────┐
             │  Mosquitto   │
             └──────┬───────┘
                    │
                    ▼
             ┌──────────────┐
             │ Subscriber   │
             │     Go       │
             └──────┬───────┘
                    │
                 /metrics
                    │
                    ▼
             ┌──────────────┐
             │  Prometheus  │
             └──────┬───────┘
                    │
                 PromQL
                    │
                    ▼
             ┌──────────────┐
             │    Grafana   │
             └──────────────┘
```

O sistema deverá permitir monitorar:

- temperatura;
- pressão;
- altitude;
- RSSI;
- memória disponível;
- uptime;
- estado reportado pelo ESP32;
- disponibilidade do dispositivo;
- histórico das métricas;
- múltiplos ESP32 simultaneamente;
- estado do próprio subscriber Go.

## Regra fundamental sobre status

A métrica:

```text
esp32_status
```

segue:

```text
OK    → 1
ERROR → 0
```

e representa **o estado reportado pelo ESP32**.

Ela **não representa diretamente se o dispositivo está online ou offline**.

A detecção de ausência de comunicação será implementada separadamente através de uma métrica de disponibilidade/último contato.