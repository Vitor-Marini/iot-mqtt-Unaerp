# grafana

Camada de visualização. Consulta o Prometheus por PromQL e desenha os painéis
da estação meteorológica.

Imagem: [`grafana/grafana:11.1.0`](https://hub.docker.com/r/grafana/grafana).

## O que faz

- Provisiona o datasource **Prometheus** no boot, com a URL vinda de variável
  de ambiente — nada de apontar o datasource pela interface e perder a
  configuração no próximo container.
- Carrega automaticamente todo `.json` de [`dashboards/`](dashboards/) na pasta
  *Estação Meteorológica*, relendo o diretório a cada 30 s.
- Persiste usuários, preferências e dashboards criados pela UI no volume
  `grafana_data`.

## Interface

| | |
|---|---|
| **Entrada** | API PromQL do Prometheus |
| **Saída** | UI web em `:3000` |
| **Depende de** | `prometheus:9090` |
| **Consumido por** | pessoas |

## Pré-requisitos

- Docker Engine 20.10+ e Docker Compose v2.
- Um `prometheus` no ar e alcançável a partir desta máquina.

## Configuração

| Variável | Descrição | Padrão |
|---|---|---|
| `GRAFANA_PORT` | Porta publicada no host | `3000` |
| `GRAFANA_ADMIN_USER` | Usuário administrador | `admin` |
| `GRAFANA_ADMIN_PASSWORD` | Senha do administrador | `admin` |
| `PROMETHEUS_URL` | URL do Prometheus | `http://prometheus:9090` |

Rodando em máquina separada, aponte para o IP real:

```bash
PROMETHEUS_URL=http://192.168.0.30:9090
```

> As credenciais padrão `admin`/`admin` servem ao laboratório desta
> disciplina. Troque `GRAFANA_ADMIN_PASSWORD` em qualquer rede compartilhada —
> o `.env` não é versionado justamente por isso.

## Como rodar

```bash
cp .env.example .env
docker compose up -d
docker compose logs -f
```

Acesse <http://localhost:3000> e entre com as credenciais do `.env`.

Parar:

```bash
docker compose down          # mantém dashboards e usuários
docker compose down -v       # zera o Grafana
```

## Como validar

```bash
# Serviço saudável?
curl -s http://localhost:3000/api/health
# {"database":"ok","version":"11.1.0",...}

# O datasource foi provisionado e responde?
curl -s -u admin:admin http://localhost:3000/api/datasources/uid/prometheus/health
# {"message":"Successfully queried the Prometheus API.","status":"OK"}
```

Na interface: **Connections → Data sources → Prometheus → Save & test**.

## Dashboards

O diretório [`dashboards/`](dashboards/) começa vazio: crie os painéis pela
interface e exporte o `.json` para lá, seguindo
[`dashboards/README.md`](dashboards/README.md). Assim eles ficam versionados e
sobem junto com o serviço em qualquer máquina.

Painéis sugeridos para a estação, com as consultas em
[`../prometheus/README.md`](../prometheus/README.md#consultas-úteis-promql):

- **Temperatura** (time series) — `weather_temperature_celsius`
- **Pressão** (time series) — `weather_pressure_hpa`
- **Altitude** (stat) — `weather_altitude_meters`
- **Status do sensor** (stat, mapeando `1`→Online e `0`→Offline) — `weather_sensor_up`
- **Sinal WiFi** (gauge) — `weather_sensor_wifi_rssi_dbm`
- **Payloads inválidos** (time series) — `rate(weather_messages_invalid_total[5m])`

## Problemas comuns

| Sintoma | Causa provável |
|---|---|
| Datasource com `Bad Gateway` | `PROMETHEUS_URL` inalcançável desta máquina |
| Datasource não aparece | Erro de sintaxe no provisioning — veja `docker compose logs grafana` |
| Painéis vazios, datasource OK | Ainda não há série `weather_*`: o subscriber não recebeu mensagens |
| Dashboard editado volta ao original | Dashboards provisionados são recarregados do arquivo; exporte e salve o `.json` |
| Senha nova não vale | `GF_SECURITY_ADMIN_PASSWORD` só é aplicada na primeira criação; use `down -v` ou troque pela UI |
