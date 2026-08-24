# Projeto desenvolvido para disciplina de Hadware configurável e IOT.

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
Acessar broker dentro do mosquito

```bash
docker exec -it mosquitto sh
```

## Subscrever no topico /temperature-data

```bash
mosquitto_sub -t /temperature-data -v
```