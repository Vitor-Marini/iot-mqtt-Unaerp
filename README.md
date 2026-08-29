# Projeto desenvolvido para disciplina de Hadware configurável e IOT.

# Tópico
/telemetry
/health-check

# Configurações do Broker
  mqtt_broker_configs = {
      "HOST": "localhost",
      "PORT": 1883,
      "CLIENT_NAME": "client_esp32",
      "KEEPALIVE": 3,
      "TOPIC": "/telemetry", "/health-check"
}

# Payload tópico
    sensor_data = {
        "sensor_id": MAC,
        "sensor_model": BMP280,
        "temperature": temperature,
        "pressure": pressure,
        "altitude": altitude,
        "timestamp": timestamp UTC,
    }


Arquitetura:
1. Publisher IOT (Raspberry ou ESP32)
2. Broker MQTT
3. Subscriber -> Banco de dados
4. Grafana conecta no banco de dados

Em loop:
- Gera dados novos via mockup
- preparar o payload do mqtt
- publica num topico especifico
- aguarda x segundos

# Instruções

## Docker
Rodar broker mqtt mosquito na porta 1883

```bash
 docker run -d \
  --name mosquitto \
  -p 1883:1883 \
  eclipse-mosquitto:2
```
## Testar broker
Acessar broker dentro do docker

```bash
docker exec -it mosquitto sh
```

Ou executar subscriber.py
```bash
uv run subscriber.py
```

## Subscrever no topico /telemetry

```bash
mosquitto_sub -t /temperature-data -v
```

## Publicar tópico 
