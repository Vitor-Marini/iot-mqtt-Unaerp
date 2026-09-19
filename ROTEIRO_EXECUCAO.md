# Roteiro de Execução e Observabilidade (Multi-Terminal)

Guia passo a passo para subir toda a infraestrutura, monitorar os serviços e disparar atualizações OTA nas 3 ESPs.

---

## 🛠️ Passo 0: Preparação Inicial (Apenas uma vez)

No diretório raiz do projeto (`/home/vitorsynkar/codes/personal/iot-mqtt-Unaerp`):

1. **Criar o arquivo `.env` da raiz:**
   ```bash
   cp .env.example .env
   ```
   Abra o `.env` e configure:
   - `OTA_EXTERNAL_HOST=192.168.10.130` *(IP da máquina no Wi-Fi)*
   - `INFLUXDB_PASSWORD=senha-com-8-digitos` *(mínimo 8 caracteres)*
   - `INFLUXDB_TOKEN=defina-um-token-seguro-aqui`

2. **Configurar o firmware das ESPs:**
   Edite `services/esp32-firmware/secrets.ini`:
   - `wifi_ssid = SeuWiFi`
   - `wifi_pass = SuaSenha`
   - `mqtt_host = 192.168.10.130`

3. **Gravar o firmware inicial nas 3 placas via USB:**
   Conecte cada uma das 3 placas via cabo USB (uma por vez) e execute:
   ```bash
   make firmware-upload
   ```

4. **Subir todos os containers Docker em segundo plano:**
   ```bash
   make up
   ```
   *(Inicia Mosquitto, InfluxDB, Subscriber, Grafana e OTA Server)*.

---

## 🖥️ Terminais de Operação e Observabilidade

Abra 5 terminais na pasta raiz do projeto:

### 📟 Terminal 1 — Logs do OTA Server
Acompanha downloads de binários pelas placas e comandos OTA enviados:
```bash
make logs S=ota-server
```

---

### 📟 Terminal 2 — Logs do Subscriber (Ingestão)
Acompanha as leituras de telemetria recebidas das ESPs e gravadas no InfluxDB:
```bash
make logs S=subscriber
```

---

### 📟 Terminal 3 — Logs do Broker MQTT (Mosquitto)
Acompanha conexões, desconexões e tráfego de mensagens MQTT:
```bash
make logs S=mosquitto
```

---

### 📟 Terminal 4 — Healthcheck Monitor (Interface Gráfica)
Abre a janela gráfica com tabela em tempo real do estado das 3 placas (IP, RSSI, heap, uptime, versão):
```bash
make monitor
```

---

### 📟 Terminal 5 — Disparo de OTA e Comandos Operacionais
Terminal livre para consultar status, compilar novas versões e disparar atualizações sem fio:

- **Verificar status de todas as placas conectadas:**
  ```bash
  cd services/ota-server && ./trigger_ota.sh status
  ```

- **Compilar nova versão de firmware:**
  ```bash
  make firmware VERSION=1.0.2
  ```

- **Disparar atualização OTA para todas as placas:**
  ```bash
  make ota-trigger VERSION=1.0.2
  ```

- **Disparar atualização OTA para uma placa específica:**
  ```bash
  make ota-trigger VERSION=1.0.2 MAC=545BA26062EC
  ```

---

## ⚡ Sequência Rápida: Compilar e Atualizar Firmware

Sempre que fizer alterações no código do firmware em `services/esp32-firmware`:

```bash
# 1. Compile definindo o número da versão (ex: 1.0.2)
make firmware VERSION=1.0.2

# 2. Suba sem fio para todas as ESPs (Broadcast):
make ota-trigger VERSION=1.0.2

# OU suba sem fio para apenas uma ESP específica (pelo MAC):
make ota-trigger VERSION=1.0.2 MAC=545BA26062EC

# (Opcional) Gravação física inicial ou recuperação via USB:
make firmware-upload
```

---

## 🌐 Acesso aos Painéis Web

- **Grafana (Visualização dos Gráficos):** [http://localhost:3000](http://localhost:3000) *(admin / admin)*
- **InfluxDB (Explorador de Dados):** [http://localhost:8086](http://localhost:8086)
