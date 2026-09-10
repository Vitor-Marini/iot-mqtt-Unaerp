# grafana

Camada de visualização. Consulta o InfluxDB por Flux e desenha os painéis da
estação meteorológica.

Imagem: [`grafana/grafana:11.1.0`](https://hub.docker.com/r/grafana/grafana).

## O que faz

- Provisiona o datasource **InfluxDB** (modo Flux) no boot, com URL, org,
  bucket e token vindos de variáveis de ambiente — nada de apontar o
  datasource pela interface e perder a configuração no próximo container.
- Carrega automaticamente todo `.json` de [`dashboards/`](dashboards/) na pasta
  *Estação Meteorológica*, relendo o diretório a cada 30 s.
- Persiste usuários, preferências e dashboards criados pela UI no volume
  `grafana_data`.

## Interface

| | |
|---|---|
| **Entrada** | API Flux do InfluxDB |
| **Saída** | UI web em `:3000` |
| **Depende de** | `influxdb:8086` |
| **Consumido por** | pessoas |

## Pré-requisitos

- Docker Engine 20.10+ e Docker Compose v2.
- Um `influxdb` no ar e alcançável a partir desta máquina, e o token dele.

## Configuração

| Variável | Descrição | Padrão |
|---|---|---|
| `GRAFANA_PORT` | Porta publicada no host | `3000` |
| `GRAFANA_ADMIN_USER` | Usuário administrador | `admin` |
| `GRAFANA_ADMIN_PASSWORD` | Senha do administrador | `admin` |
| `INFLUXDB_URL` | URL do InfluxDB | `http://influxdb:8086` |
| `INFLUXDB_ORG` | Organização do InfluxDB | `esp32` |
| `INFLUXDB_BUCKET` | Bucket padrão das consultas | `sensors` |
| `INFLUXDB_TOKEN` | Token de leitura | **obrigatório** |

Rodando em máquina separada, aponte para o IP real:

```bash
INFLUXDB_URL=http://192.168.0.30:8086
```

`INFLUXDB_TOKEN` não tem padrão: o compose falha na hora se estiver vazio, em
vez de subir um Grafana com datasource quebrado. Use o mesmo valor do
[`services/influxdb/.env`](../influxdb/.env.example).

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
curl -s -u admin:admin http://localhost:3000/api/datasources/uid/influxdb/health
# {"message":"datasource is working","status":"OK"}
```

Na interface: **Connections → Data sources → InfluxDB → Save & test**.

## Dashboards

O diretório [`dashboards/`](dashboards/) começa vazio: crie os painéis pela
interface e exporte o `.json` para lá, seguindo
[`dashboards/README.md`](dashboards/README.md). Assim eles ficam versionados e
sobem junto com o serviço em qualquer máquina.

Painéis sugeridos para a estação. As consultas Flux prontas estão em
[`../influxdb/README.md`](../influxdb/README.md#consultas-úteis-flux):

| Painel | Tipo | Measurement / field |
|---|---|---|
| Temperatura | time series | `telemetry` / `temperature` |
| Pressão | time series | `telemetry` / `pressure` |
| Altitude | stat | `telemetry` / `altitude` |
| Estado do sensor | stat (`1`→OK, `0`→ERROR) | `healthcheck` / `status` |
| Sinal WiFi | gauge | `healthcheck` / `rssi` |
| Heap livre | time series | `healthcheck` / `free_heap` |
| Silêncio desde o último contato | stat | `healthcheck`, ver consulta de queda |

Agrupe por `sensor_id` nos painéis para separar múltiplos ESP32.

## Problemas comuns

| Sintoma | Causa provável |
|---|---|
| Datasource com `Bad Gateway` | `INFLUXDB_URL` inalcançável desta máquina |
| Datasource não aparece | Erro de sintaxe no provisioning — veja `docker compose logs grafana` |
| Datasource com `unauthorized` | `INFLUXDB_TOKEN` diferente do configurado no InfluxDB |
| Painéis vazios, datasource OK | O bucket está vazio: o subscriber não recebeu mensagens |
| Dashboard editado volta ao original | Dashboards provisionados são recarregados do arquivo; exporte e salve o `.json` |
| Senha nova não vale | `GF_SECURITY_ADMIN_PASSWORD` só é aplicada na primeira criação; use `down -v` ou troque pela UI |
