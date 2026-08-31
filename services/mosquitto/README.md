# mosquitto

Broker MQTT do projeto. É o ponto de encontro entre o ESP32 (publicador) e o
subscriber Go (assinante) — nenhum dos dois se conhece diretamente.

Imagem: [`eclipse-mosquitto:2.0`](https://hub.docker.com/_/eclipse-mosquitto).

## O que faz

- Aceita conexões MQTT 3.1.1 em `0.0.0.0:1883`.
- Roteia `devices/{MAC}/telemetry` e `devices/{MAC}/health-check` (ver
  [`docs/mqtt-contract.md`](../../docs/mqtt-contract.md)).
- Guarda as mensagens **retidas** de cada dispositivo, então quem assina
  recebe de imediato a última leitura conhecida de cada um.
- Persiste sessões QoS 1 e mensagens retained em volume, sobrevivendo a
  reinícios do container.

## Interface

| | |
|---|---|
| **Entrada** | conexões MQTT TCP em `1883` |
| **Saída** | entrega aos assinantes |
| **Depende de** | nada |
| **Consumido por** | `esp32-firmware` (publish), `subscriber` (subscribe) |

## Pré-requisitos

- Docker Engine 20.10+ e Docker Compose v2.
- Porta `1883` liberada no firewall da máquina — o ESP32 e o subscriber vêm
  de outros hosts.

## Configuração

| Variável | Descrição | Padrão |
|---|---|---|
| `MQTT_PORT` | Porta publicada no host | `1883` |

O restante fica em [`config/mosquitto.conf`](config/mosquitto.conf), montado
como somente-leitura no container.

## Como rodar

```bash
docker compose up -d
docker compose logs -f
```

Parar:

```bash
docker compose down          # mantém os volumes
docker compose down -v       # apaga dados e logs persistidos
```

## Como validar

O broker sobe em segundos. Confirme o healthcheck:

```bash
docker compose ps
# STATUS deve mostrar "Up (healthy)"
```

Teste ponta a ponta de dentro do container (não precisa instalar nada no host):

```bash
# Terminal 1 — assinar
docker compose exec mosquitto mosquitto_sub -t 'devices/+/telemetry' -v

# Terminal 2 — publicar uma leitura de teste
docker compose exec mosquitto mosquitto_pub -t 'devices/A1B2C3D4E5F6/telemetry' -m \
  '{"sensor_id":"A1B2C3D4E5F6","sensor_model":"BMP280","temperature":24.5,"pressure":1013.25,"altitude":540.2,"timestamp":1787960400}'
```

O terminal 1 deve imprimir a mensagem. Isso valida o broker sem depender do
ESP32 nem do subscriber.

Ver o estado atual da estação (mensagem retained):

```bash
docker compose exec mosquitto mosquitto_sub -t 'devices/+/health-check' -v -C 1
```

Estatísticas internas do broker:

```bash
docker compose exec mosquitto mosquitto_sub -t '$SYS/broker/clients/connected' -v -C 1
```

## Segurança

Este broker roda com `allow_anonymous true`. **É uma decisão deliberada para o
ambiente de laboratório** desta disciplina: mantém o firmware e o subscriber
sem credenciais e o foco no pipeline de dados.

Se for expor o broker a uma rede não confiável, habilite autenticação:

```bash
# 1. Criar o arquivo de senhas
docker compose exec mosquitto mosquitto_passwd -c -b \
  /mosquitto/config/password.txt estacao uma-senha-forte

# 2. Em config/mosquitto.conf, trocar:
#      allow_anonymous false
#      password_file /mosquitto/config/password.txt

# 3. Recarregar
docker compose restart
```

Depois preencha `MQTT_USERNAME`/`MQTT_PASSWORD` no `config.h` do firmware e no
`.env` do subscriber. Para TLS, adicione um `listener 8883` com `cafile`,
`certfile` e `keyfile`.

## Problemas comuns

| Sintoma | Causa provável |
|---|---|
| `Error: Address already in use` | Já existe um Mosquitto (ou outro serviço) na 1883 no host |
| ESP32 não conecta, mas `mosquitto_sub` local funciona | Firewall do host bloqueando a 1883 vinda da rede |
| `Connection Refused: not authorised` | `allow_anonymous false` sem credenciais configuradas nos clientes |
| Logs param após reiniciar | Volume `mosquitto_log` removido com `down -v` |
