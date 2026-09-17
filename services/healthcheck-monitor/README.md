# healthcheck-monitor

Segundo consumidor MQTT do projeto. Assina o tópico de health check e mostra o
estado de cada ESP32 numa **janela Tkinter** — não grava nada.

É o par do [`subscriber`](../subscriber/): os dois leem o mesmo broker, com
propósitos opostos. O subscriber persiste no InfluxDB para histórico e
dashboards; este serve para olhar agora e saber se as placas estão vivas, sem
depender de banco nem de Grafana no ar.

## O que faz

- Assina `devices/+/health-check`. O wildcard é necessário porque o MAC do
  dispositivo faz parte do tópico.
- Mantém uma linha por dispositivo na tabela, atualizada a cada mensagem: MAC,
  nome, modelo, status, RSSI, heap livre, uptime e timestamp.
- A identidade vem do **segmento do tópico**, não do payload — então funciona
  também com firmware anterior ao rename de `sensor_id` para `device_id`.
- Não guarda histórico e não escreve em disco. Fechou a janela, acabou.

Formato das mensagens em [`docs/mqtt-contract.md`](../../docs/mqtt-contract.md);
a fonte da verdade é o firmware.

## Interface

| | |
|---|---|
| **Entrada** | MQTT `devices/+/health-check` |
| **Saída** | janela Tkinter |
| **Depende de** | `mosquitto:1883` e um display gráfico |
| **Consumido por** | pessoas |

Não abre porta nenhuma.

## Como rodar

### No host — recomendado

É uma aplicação de janela: rodar direto no sistema é o caminho simples.

```bash
cd services/healthcheck-monitor
pip install -r requirements.txt
cp .env.example .env          # MQTT_HOST=localhost se o broker for local
python monitor.py
```

Ou sem editar arquivo:

```bash
MQTT_HOST=192.168.10.50 python monitor.py
```

O `tkinter` acompanha o Python oficial no Windows e no macOS. No Linux costuma
ser um pacote à parte:

```bash
sudo apt install python3-tk      # Debian/Ubuntu
sudo dnf install python3-tkinter # Fedora
```

### Em container — precisa de servidor X

O `Dockerfile` e o `docker-compose.yml` existem e funcionam, mas **uma GUI em
container precisa de um servidor X alcançável pelo `DISPLAY`**. Não é o caminho
mais curto; use se quiser o serviço empacotado como os outros.

No compose da raiz este serviço está no profile `gui`, então **não sobe** com
`make up` — o que evita um container inútil quando não há display configurado:

```bash
docker compose --profile gui up -d healthcheck-monitor
```

#### Linux

O socket X do host já é montado pelo compose. Basta liberar o acesso:

```bash
xhost +local:docker
DISPLAY=:0 docker compose up --build
```

#### Windows

Precisa de um servidor X no host — [VcXsrv](https://sourceforge.net/projects/vcxsrv/)
é o usual. Marque *Disable access control* ao iniciar o XLaunch, e então:

```powershell
$env:DISPLAY = "host.docker.internal:0.0"
docker compose up --build
```

#### macOS

Com [XQuartz](https://www.xquartz.org/) instalado:

```bash
xhost + 127.0.0.1
DISPLAY=host.docker.internal:0 docker compose up --build
```

Se a janela não aparecer, o erro costuma ser
`couldn't connect to display` no log — é o `DISPLAY` errado ou o servidor X
sem permitir a conexão.

## Configuração

| Variável | Descrição | Padrão |
|---|---|---|
| `MQTT_HOST` | Host do broker | `localhost` |
| `MQTT_PORT` | Porta do broker | `1883` |
| `MQTT_HEALTH_TOPIC` | Filtro de tópico | `devices/+/health-check` |
| `MONITOR_CLIENT_ID` | Client ID no broker | `healthcheck-monitor` |
| `DISPLAY` | Só em container: onde desenhar a janela | `:0` |

O `MONITOR_CLIENT_ID` **precisa ser diferente** do `SUBSCRIBER_ID` do
subscriber Go: dois clientes MQTT com o mesmo ID se expulsam mutuamente do
broker, num laço de reconexão.

### Qual valor de `MQTT_HOST`

Depende de onde o broker está em relação a este processo:

| Este processo | Broker | `MQTT_HOST` |
|---|---|---|
| host (`python monitor.py`) | mesma máquina | `localhost` |
| host | outra máquina | IP dela |
| container, compose próprio | mesma máquina | `host.docker.internal` |
| container | outra máquina | IP dela |
| container, compose da raiz | mesmo compose | `mosquitto` |

`localhost` **nunca** funciona de dentro de um container. Detalhes e o caso
Linux × Windows em [`docs/deployment.md`](../../docs/deployment.md).

## Como validar

Com o broker no ar, publique um health check no formato real:

```bash
mosquitto_pub -h localhost -t "devices/A1B2C3D4E5F6/health-check" -m \
  '{"device_id":"A1B2C3D4E5F6","device_name":"Estacao-Lab","sensor_model":"BMP280","version":"1.1.0","status":"OK","ip":"192.168.0.31","rssi":-65,"free_heap":215400,"uptime_ms":45000,"timestamp":1787960400}'
```

A linha deve aparecer na tabela de imediato, e o rodapé mostrar
`Dispositivos ativos: 1`. Publique em outro tópico `devices/{MAC}/health-check`
e surge uma segunda linha; republique no mesmo tópico e a linha existente é
atualizada no lugar.

O [`mock_esp32.sh`](../subscriber/mock_esp32.sh) do subscriber também alimenta
este monitor: ele publica nos dois tópicos.

## Limitações conhecidas

São do desenho atual, não defeitos a corrigir às pressas:

- **Sem reconexão automática.** Se o broker cair, a janela continua aberta com
  os últimos valores e não volta sozinha. Em container, `restart:
  unless-stopped` derruba e sobe o processo; no host, é reabrir.
- **Se o broker estiver fora no start, o processo falha.** `client.connect()`
  levanta exceção antes de a janela abrir.
- **Nada indica dispositivo offline.** A linha fica com os últimos valores
  indefinidamente. Como o firmware não registra LWT, detectar queda exigiria
  comparar o horário da última mensagem — hoje não é feito.
- **Payload inválido é ignorado em silêncio.** O `except Exception: pass` do
  `on_message` descarta sem avisar.

## Problemas comuns

| Sintoma | Causa provável |
|---|---|
| `ModuleNotFoundError: No module named 'tkinter'` | Falta `python3-tk` (Linux) |
| `couldn't connect to display` | Em container sem servidor X, ou `DISPLAY` errado |
| `ConnectionRefusedError` no start | `MQTT_HOST` errado — ver a tabela de endereços |
| Tabela vazia, sem erro | Tópico divergente, ou nenhum ESP32 publicando health check |
| Conecta e cai em laço | `MONITOR_CLIENT_ID` igual ao de outro cliente MQTT |
| Coluna `Dispositivo` mostrando o MAC | A placa ainda não recebeu o firmware que publica `device_name` |
| `timestamp` mostrando número pequeno | O ESP32 não sincronizou NTP e mandou o uptime; ver o [contrato](../../docs/mqtt-contract.md) |
