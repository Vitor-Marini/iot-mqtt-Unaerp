# OTA Server

Serviço responsável pelo armazenamento, distribuição e orquestração de atualizações de firmware **Over-The-Air (OTA)** para a frota de microcontroladores **ESP32**.

Desenvolvido em **Go**, o serviço fornece:
1. **Servidor HTTP:** Entrega os binários compilados (`.bin`) em streaming com cálculo e envio de checksum `x-MD5`.
2. **Orquestrador MQTT:** Dispara comandos de atualização em lote (broadcast) ou direcionados por nó, e escuta em tempo real as confirmações de status emitidas pelas placas.
3. **CLI Interativo:** Script `./trigger_ota.sh` para disparar atualizações e acompanhar o status diretamente pelo terminal.

---

## 🚀 Arquitetura & Fluxo

```
[Desenvolvedor / CI-CD]
         │
         ▼ copia firmware.bin
 ┌──────────────┐
 │  ota-server  │ ────► Serve HTTP GET /firmware/latest.bin (Porta 8080)
 │     (Go)     │ ────► Publica comando OTA via MQTT (Porta 1883)
 └──────┬───────┘
        │ MQTT
        ▼
 ┌──────────────┐
 │  mosquitto   │ ────► Broadcast: devices/broadcast
 └──────┬───────┘ ────► Unicast:   devices/{MAC}/commands
        │
        ▼ MQTT
 ┌──────────────┐
 │  Frota ESP32 │ ────► Baixa binário via HTTP, grava na flash, reinicia
 └──────────────┘ ────► Publica status em devices/{MAC}/ota-status
```

---

## 🛠️ Configuração

Copie o arquivo de exemplo:
```bash
cp .env.example .env
```

| Variável | Padrão | Descrição |
|---|---|---|
| `OTA_PORT` | `8080` | Porta HTTP onde o servidor escuta |
| `OTA_EXTERNAL_HOST` | `192.168.1.100` | IP do host acessível pelos ESP32 na rede WiFi |
| `STORAGE_DIR` | `./storage` | Pasta onde ficam os binários `.bin` |
| `MQTT_HOST` | `localhost` | Host do Broker Mosquitto |
| `MQTT_PORT` | `1883` | Porta do Broker Mosquitto |
| `MQTT_TOPIC_BASE` | `devices` | Prefixo base dos tópicos MQTT |

---

## 📦 Como Usar

### 1. Compilar ou Copiar o Firmware para `storage/`
Ao compilar o firmware no PlatformIO, o binário é gerado em:
`services/esp32-firmware/.pio/build/esp32dev/firmware.bin`

Copie para o diretório `storage/`:
```bash
cp ../esp32-firmware/.pio/build/esp32dev/firmware.bin ./storage/firmware-v1.0.1.bin
```

### 2. Rodar o Servidor
Localmente com Go:
```bash
make run
```
Ou com Docker:
```bash
make up
```

### 3. Disparar a Atualização pelo Terminal
Para disparar para todas as placas da rede:
```bash
./trigger_ota.sh --target all --version 1.0.1
```

Para disparar para uma placa específica:
```bash
./trigger_ota.sh --target A1B2C3D4E5F6 --version 1.0.1
```

O script exibirá o progresso e confirmações em tempo real:
```text
[ESP32 STATUS] 📥 Nó [A1B2C3D4E5F6] (192.168.1.150): Baixando firmware versao 1.0.1...
[HTTP STREAM] 📦 Iniciando envio de 'firmware-v1.0.1.bin' (974336 bytes, MD5: ...) para 192.168.1.150...
[HTTP STREAM] ✅ Concluida transferencia de 'firmware-v1.0.1.bin' para 192.168.1.150
[ESP32 STATUS] 💾 Nó [A1B2C3D4E5F6]: Gravando na memoria Flash...
[ESP32 STATUS] ✅ SUCESSO! Nó [A1B2C3D4E5F6] reiniciou e validou o novo firmware v1.0.1!
```

---

## 🌐 Endpoints HTTP

- `GET /health` — Status de saúde da aplicação.
- `GET /firmware/{arquivo}` — Download em streaming com header `x-MD5`.
- `GET /api/firmware/latest` — Metadados do binário mais recente em `storage/`.
- `POST /api/ota/publish` — Envia comando MQTT de atualização.
  - Body: `{"target": "all", "version": "1.0.1", "filename": "firmware-v1.0.1.bin"}`
- `GET /api/nodes/status` — Retorna o estado atual de cada ESP32 na memória do servidor.
