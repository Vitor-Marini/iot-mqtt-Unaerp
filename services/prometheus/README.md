# prometheus

Banco de dados temporal do projeto. Faz scrape do endpoint `/metrics` do
`subscriber` a cada 15 s e guarda a série histórica que o Grafana consulta.

Imagem: [`prom/prometheus:v2.53.0`](https://hub.docker.com/r/prom/prometheus).

## O que faz

- Raspa `subscriber:2112/metrics` e o próprio `localhost:9090/metrics`.
- Armazena as séries em TSDB local, com retenção padrão de 15 dias.
- Expõe a API de consulta PromQL em `:9090`, usada pelo Grafana.
- Registra `up{job="subscriber"}` automaticamente — se o subscriber cair, isso
  aparece na série sem precisar de código nenhum.

## Interface

| | |
|---|---|
| **Entrada** | scrape HTTP do `subscriber` |
| **Saída** | API PromQL em `:9090` + UI web |
| **Depende de** | `subscriber:2112` |
| **Consumido por** | `grafana` |

## Pré-requisitos

- Docker Engine 20.10+ e Docker Compose v2.
- Porta `9090` alcançável pela máquina do Grafana.
- O `subscriber` no ar e alcançável a partir desta máquina.

## Configuração

| Variável | Descrição | Padrão |
|---|---|---|
| `PROMETHEUS_PORT` | Porta publicada no host | `9090` |
| `PROMETHEUS_RETENTION` | Janela de retenção do TSDB | `15d` |

### Apontar para o subscriber

Como o subscriber roda em outra máquina, o alvo **não** fica no
`prometheus.yml`, e sim em [`config/targets/subscriber.json`](config/targets/subscriber.json).
O Prometheus relê esse arquivo a cada 30 s, então trocar o IP não exige
reiniciar nada:

```jsonc
[
  {
    "targets": ["192.168.0.20:2112"],
    "labels": { "servico": "subscriber", "ambiente": "producao" }
  }
]
```

Se editar o `prometheus.yml` (que exige recarga), use:

```bash
curl -X POST http://localhost:9090/-/reload
```

## Como rodar

```bash
cp .env.example .env
docker compose up -d
docker compose logs -f
```

Parar:

```bash
docker compose down          # mantém o histórico
docker compose down -v       # apaga o TSDB
```

## Como validar

```bash
# Serviço saudável?
curl -s http://localhost:9090/-/healthy

# A configuração foi aceita?
curl -s http://localhost:9090/api/v1/status/config | head -c 200
```

Confira os alvos em <http://localhost:9090/targets> — o job `subscriber` deve
estar **UP**. Pela linha de comando:

```bash
curl -s 'http://localhost:9090/api/v1/query?query=up{job="subscriber"}'
```

Consulta de exemplo, já com dados fluindo:

```bash
curl -s --get http://localhost:9090/api/v1/query \
  --data-urlencode 'query=weather_temperature_celsius'
```

## Consultas úteis (PromQL)

| Objetivo | Consulta |
|---|---|
| Temperatura atual | `weather_temperature_celsius` |
| Média de 5 min | `avg_over_time(weather_temperature_celsius[5m])` |
| Variação de pressão por hora | `deriv(weather_pressure_hpa[1h]) * 3600` |
| Sensores offline | `weather_sensor_up == 0` |
| Taxa de mensagens | `rate(weather_messages_received_total[5m])` |
| % de payloads inválidos | `100 * rate(weather_messages_invalid_total[5m]) / rate(weather_messages_received_total[5m])` |
| Silêncio do sensor (s) | `time() - weather_sensor_last_seen_timestamp_seconds` |

## Problemas comuns

| Sintoma | Causa provável |
|---|---|
| Alvo `DOWN` com `connection refused` | IP errado em `targets/subscriber.json`, ou firewall na 2112 |
| Alvo `DOWN` com `context deadline exceeded` | `scrape_timeout` menor que a resposta do subscriber |
| Nenhuma série `weather_*` | O subscriber está no ar mas nunca recebeu mensagem MQTT |
| Histórico some ao reiniciar | Volume `prometheus_data` removido com `down -v` |
| Disco enchendo | Reduza `PROMETHEUS_RETENTION` |
