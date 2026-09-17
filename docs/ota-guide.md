# ⚡ Guia Rápido: Atualização OTA ESP32 (Cheat Sheet)

Copie e cole os comandos abaixo para compilar e atualizar sua frota em menos de 1 minuto:

---

### 1️⃣ Compilar o Firmware com a Nova Versão
```bash
cd services/esp32-firmware
make compile VERSION=1.0.5
```
*(Gera o arquivo binário com a versão desejada e copia direto para `services/ota-server/storage/firmware.bin`)*.

---

### 2️⃣ Iniciar o Servidor OTA
*(Em um terminal dedicado)*:
```bash
cd services/ota-server
make run
```
*(O servidor detecta automaticamente o IP da sua placa Wi-Fi no roteador e conecta ao Mosquitto)*.

---

### 3️⃣ Disparar a Atualização para Todas as Placas
*(Em outro terminal)*:
```bash
cd services/ota-server
./trigger_ota.sh 1.0.5
```

---

### 🔍 Comandos Extras Úteis:
- **Verificar o status atual de todas as placas conectadas:**
  ```bash
  ./trigger_ota.sh status
  ```
- **Atualizar apenas uma placa específica pelo endereço MAC:**
  ```bash
  ./trigger_ota.sh 1.0.5 6893BED5D8C4
  ```
- **Forçar um IP específico se estiver em redes múltiplas:**
  ```bash
  make run HOST=192.168.10.130
  ```

---

## 📌 Como Funciona por Trás dos Panos
1. O comando `make compile VERSION=...` compila o C++ com `-D FIRMWARE_VERSION="X.X.X"` e coloca em `services/ota-server/storage/firmware.bin`.
2. O `./trigger_ota.sh X.X.X` publica uma ordem MQTT no tópico `devices/broadcast`.
3. Todas as placas ESP32 recebem o link HTTP `http://<SEU_IP>:8080/firmware/firmware.bin`.
4. Cada placa baixa o binário, grava na partição OTA da memória Flash, reinicia e começa a rodar a nova versão!
