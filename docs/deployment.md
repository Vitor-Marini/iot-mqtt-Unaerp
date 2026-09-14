# Implantação

Este documento existe porque a maior parte do tempo perdido neste projeto não
foi escrevendo código: foi descobrindo **qual endereço colocar em cada `.env`**.
A regra é simples quando explicitada, e está na seção seguinte.

## As três formas de rodar

| Modo | Onde os serviços ficam | Compose a usar |
|---|---|---|
| **Full local** | tudo numa máquina | `docker-compose.yml` da raiz |
| **Dividido** | 2+ máquinas, cada uma com um subconjunto | um `services/*/docker-compose.yml` por serviço |
| **Um por máquina** | uma máquina por serviço | um `services/*/docker-compose.yml` por serviço |

O modo **dividido** é o que usamos de fato, e é o que originalmente faltava aqui.
Ele não é um modo novo: é o mesmo dos composes por serviço, só que com alguns
serviços compartilhando máquina. Toda a dificuldade está nos endereços.

## A única pergunta que importa

Antes de preencher qualquer `.env`, responda: **onde está o alvo, em relação a
quem vai falar com ele?**

| Quem fala | Onde está o alvo | Endereço a usar |
|---|---|---|
| container | mesmo compose (stack da raiz) | **nome do serviço** — `mosquitto`, `influxdb` |
| container | outro compose, **mesma máquina** | **`host.docker.internal`** |
| container | **outra máquina** | **IP daquela máquina** — `192.168.0.10` |
| processo no host (`go run`, `python monitor.py`) | mesma máquina | **`localhost`** |
| processo no host | outra máquina | IP daquela máquina |
| ESP32 (fora do Docker) | máquina do broker | IP daquela máquina |

Três consequências que custaram tempo:

- **`localhost` dentro de um container nunca funciona.** Ali `localhost` é o
  próprio container, não a máquina. Um subscriber em container com
  `MQTT_HOST=localhost` tenta conectar em si mesmo.
- **Nome de serviço só resolve dentro do mesmo compose.** Composes separados
  criam redes separadas (`mosquitto_default`, `subscriber_default`), e o DNS não
  atravessa. `container_name` também não ajuda.
- **`host.docker.internal` funciona porque a porta está publicada.** O container
  sai até o host e volta pela porta mapeada. Se o serviço alvo não publicar
  porta, não há por onde entrar.

## Linux × Windows

`host.docker.internal` existe nativamente só no Docker Desktop (Windows e
macOS). Em Docker nativo no Linux, o nome não existe — e foi exatamente isso que
quebrou quando a máquina A era Linux e a B era Windows.

Os composes deste repo resolvem isso declarando:

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

`host-gateway` é resolvido pelo próprio Docker para o gateway do host (no Linux,
o `172.17.0.1` da bridge). Com isso, **o mesmo valor de `.env` funciona nos dois
sistemas** — que era o objetivo.

Está declarado nos três serviços que fazem conexão de saída:

| Serviço | Tem `extra_hosts` | Por quê |
|---|---|---|
| `subscriber` | sim | fala com mosquitto e influxdb |
| `grafana` | sim | fala com influxdb |
| `healthcheck-monitor` | sim | fala com mosquitto (quando em container) |
| `mosquitto` | não | só recebe conexões |
| `influxdb` | não | só recebe conexões |

Requisitos e ressalvas:

- **Docker 20.10+.** `host-gateway` não existe antes disso.
- **A porta precisa estar publicada em `0.0.0.0`**, não em `127.0.0.1`. Os
  composes deste repo já fazem isso.
- **Firewall do host no Linux.** `ufw` e `firewalld` podem barrar tráfego vindo
  da bridge do Docker. Se `host.docker.internal` resolver mas a conexão for
  recusada, libere a porta para a sub-rede `172.17.0.0/16`.

### Alternativa: rede externa compartilhada

Para serviços na mesma máquina em composes separados, existe uma opção mais
"limpa" que não passa pelo host: uma rede Docker externa compartilhada.

```bash
docker network create iot-estacao
```

E em cada compose, o serviço entra nessa rede. Aí os nomes (`mosquitto`,
`influxdb`) resolvem direto entre containers, sem hairpin pelo host.

**Não adotamos** porque adiciona um pré-requisito manual antes de qualquer
`docker compose up`, e o ganho sobre `host.docker.internal` é pequeno nesta
escala. Fica registrado como caminho válido se a comunicação host-bridge virar
problema.

## Cenário dividido: o exemplo real

Máquina **A** (Linux) roda broker e subscriber. Máquina **B** (Windows) roda
banco, dashboards e monitor. Suponha `A = 192.168.10.50` e `B = 192.168.10.137`.

### Máquina A — `services/mosquitto/`

Sem `.env`. Só subir:

```bash
cd services/mosquitto && docker compose up -d
```

### Máquina A — `services/subscriber/.env`

```env
# Broker na MESMA máquina, em outro compose:
MQTT_HOST=host.docker.internal
MQTT_PORT=1883
SUBSCRIBER_ID=DB_SUBSCRIBER

# InfluxDB na máquina B:
INFLUX_HOST=http://192.168.10.137:8086
TOKEN_INFLUX=<o mesmo token da máquina B>
INFLUX_ORG=esp32
INFLUX_BUCKET=sensors
```

### Máquina B — `services/influxdb/.env`

```env
INFLUXDB_PORT=8086
INFLUXDB_USERNAME=admin
INFLUXDB_PASSWORD=senha-com-8-ou-mais   # menos que isso = crash loop
INFLUXDB_ORG=esp32
INFLUXDB_BUCKET=sensors
INFLUXDB_RETENTION=0
INFLUXDB_TOKEN=<gere um e use o mesmo nos três .env>
```

### Máquina B — `services/grafana/.env`

```env
GRAFANA_PORT=3000
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin

# InfluxDB na MESMA máquina, em outro compose:
INFLUXDB_URL=http://host.docker.internal:8086
INFLUXDB_ORG=esp32
INFLUXDB_BUCKET=sensors
INFLUXDB_TOKEN=<o mesmo token>
```

### Máquina B — `services/healthcheck-monitor/.env`

Este é uma janela Tkinter, e roda melhor **direto no host** — sem Docker, sem
servidor X para configurar:

```env
# Broker na máquina A:
MQTT_HOST=192.168.10.50
MQTT_PORT=1883
MQTT_HEALTH_TOPIC=devices/+/health-check
MONITOR_CLIENT_ID=healthcheck-monitor
```

```bash
cd services/healthcheck-monitor
pip install -r requirements.txt
python monitor.py
```

Em container, some a vantagem: seria preciso um servidor X alcançável pelo
`DISPLAY` (VcXsrv no Windows, socket X no Linux). O README do serviço tem os
comandos, se quiserem esse caminho.

### ESP32 — `services/esp32-firmware/secrets.ini`

```ini
mqtt_host = 192.168.10.50    ; IP da máquina A, onde está o broker
```

### Ordem de subida

O InfluxDB precisa estar provisionado antes do subscriber escrever, e o broker
antes de qualquer assinante:

```
B: influxdb  →  A: mosquitto  →  A: subscriber  →  B: grafana  →  B: monitor  →  ESP32
```

## Firewall

Portas que precisam aceitar conexão de fora da máquina:

| Máquina | Porta | Quem precisa alcançar |
|---|---|---|
| onde está o `mosquitto` | `1883` | ESP32, subscriber e monitor de outras máquinas |
| onde está o `influxdb` | `8086` | subscriber e grafana de outras máquinas, navegador |
| onde está o `grafana` | `3000` | navegador |

`subscriber` e `healthcheck-monitor` não escutam em porta nenhuma: nada precisa
alcançá-los, e não há nada a liberar.

No Windows, a primeira execução do Docker Desktop costuma abrir um diálogo do
Firewall — se você negou sem perceber, a porta fica inacessível de fora mesmo
com o container no ar.

## Diagnóstico

Comece pelo tipo de erro, que já elimina metade das possibilidades:

| Erro | Significa | Olhe para |
|---|---|---|
| `connection refused` | a máquina respondeu, mas nada escuta naquela porta | o container está de pé? `docker compose ps` |
| `connection timed out` / `no route to host` | nem chegou | IP errado, máquinas em redes diferentes, firewall |
| `no such host` | o nome não resolveu | `mosquitto` fora do compose da raiz, ou `host.docker.internal` em Linux sem `extra_hosts` |
| `unauthorized access` | chegou e conectou | token divergente entre os `.env` |
| `bucket not found` | chegou e autenticou | `INFLUX_BUCKET` diferente do provisionado |

Roteiro rápido:

```bash
# 1. Todos de pé e saudáveis? Um "Restarting" aqui explica connection refused.
docker compose ps

# 2. Por que reiniciou?
docker compose logs --tail 30 <servico>

# 3. O alvo responde do host?
curl http://localhost:8086/health          # influxdb, na própria máquina
curl http://192.168.10.137:8086/health     # influxdb, de outra máquina

# 4. O alvo responde de DENTRO de um container?
docker run --rm --add-host host.docker.internal:host-gateway alpine:3.20 \
  wget -qO- http://host.docker.internal:8086/health

# 5. O broker entrega mensagem?
docker compose exec mosquitto mosquitto_sub -t 'devices/+/health-check' -v
```

## Armadilhas que já nos pegaram

Todas reais, todas neste projeto:

- **`env file .env not found`** ao rodar `docker compose up`. O `.env` não é
  versionado (tem token dentro), então quem clona só recebe o `.env.example`.
  O compose aborta antes de construir qualquer coisa. Sempre
  `cp .env.example .env` primeiro.
- **InfluxDB em crash loop com senha curta.** `INFLUXDB_PASSWORD` precisa de
  8 caracteres ou mais. Com menos, o setup falha, o container morre, reinicia e
  falha de novo — e todo mundo recebe `connection refused` na 8086. O log diz
  `passwords must be between 8 and 72 characters long`.
- **Provisionamento do InfluxDB roda uma vez só.** Senha, org, bucket e token
  são aplicados apenas quando o volume está vazio. Trocar no `.env` depois não
  muda nada: use a UI, ou `docker compose down -v` (que apaga o histórico).
- **IP de DHCP muda ao trocar de rede.** Sair da rede de testes invalida todo
  `.env` que tenha IP. Para links dentro da mesma máquina, prefira
  `host.docker.internal`, que não depende de rede. Para links entre máquinas,
  reserve os IPs no roteador.
- **`nome.local` não resolve de dentro de container.** O Windows resolve mDNS,
  mas a imagem Linux do Grafana ou do monitor não tem resolvedor mDNS. Não tente
  substituir IP por hostname assim.
- **Token precisa ser idêntico em três `.env`.** E o nome da variável muda:
  `INFLUXDB_TOKEN` no influxdb e no grafana, **`TOKEN_INFLUX`** no subscriber.
- **Tópico é `health-check`, com hífen.** O subscriber Go já assinou
  `healthcheck` por um tempo e nunca recebeu nada.
- **`Aviso: arquivo .env não encontrado` no log do subscriber é inofensivo
  dentro do Docker.** O arquivo está no `.dockerignore` de propósito (tem
  token); o compose injeta os valores como variáveis de ambiente, e o código lê
  do ambiente. A configuração chega certa apesar do aviso.
- **Dois clientes MQTT com o mesmo Client ID se expulsam.** `SUBSCRIBER_ID` e
  `MONITOR_CLIENT_ID` precisam ser diferentes — é por isso que são duas
  variáveis.
- **GUI em container precisa de servidor X.** O `healthcheck-monitor` é uma
  janela Tkinter. No compose da raiz ele está no profile `gui` e não sobe com
  `make up`, justamente para não deixar um container inútil de pé. Rodar no host
  é o caminho curto.
