"""healthcheck-monitor — janela Tkinter com o estado dos dispositivos.

Segundo consumidor MQTT do projeto. Diferente do subscriber Go, este não grava
nada: só assina o tópico de health check e mostra o último estado de cada ESP32
numa tabela.

Configuração por variável de ambiente — ver .env.example.
"""

import json
import os
import tkinter as tk
from tkinter import ttk

import paho.mqtt.client as mqtt
from paho.mqtt.enums import CallbackAPIVersion

BROKER = os.environ.get("MQTT_HOST", "localhost")
PORT = int(os.environ.get("MQTT_PORT", "1883"))
# Wildcard e hífen: o MAC do dispositivo faz parte do tópico e o sufixo é
# `health-check`, conforme docs/mqtt-contract.md. Com `/healthcheck` nada chega.
TOPIC = os.environ.get("MQTT_HEALTH_TOPIC", "devices/+/health-check")
# Client ID precisa diferir do SUBSCRIBER_ID do subscriber Go: dois clientes
# com o mesmo ID se expulsam mutuamente do broker.
CLIENT_ID = os.environ.get("MONITOR_CLIENT_ID", "healthcheck-monitor")


class MultiHealthcheckApp:
    def __init__(self, root):
        self.root = root
        self.root.title("Monitoramento Multi-Dispositivo IoT - Healthcheck")
        self.root.geometry("820x350")

        # Título superior
        title = tk.Label(
            root,
            text="DISPOSITIVOS MONITORADOS",
            font=("Arial", 13, "bold"),
            bg="#2c3e50",
            fg="white",
            pady=8,
        )
        title.pack(fill=tk.X)

        # Frame da Tabela
        table_frame = ttk.Frame(root, padding=10)
        table_frame.pack(fill=tk.BOTH, expand=True)

        columns = (
            "device_id",
            "device_name",
            "model",
            "status",
            "rssi",
            "free_heap",
            "uptime",
            "timestamp",
        )
        self.tree = ttk.Treeview(table_frame, columns=columns, show="headings", height=8)

        # Configuração das Colunas
        headers = {
            "device_id": "MAC",
            "device_name": "Dispositivo",
            "model": "Modelo",
            "status": "Status",
            "rssi": "Sinal (RSSI)",
            "free_heap": "Free Heap",
            "uptime": "Uptime",
            "timestamp": "Timestamp",
        }

        col_widths = {
            "device_id": 110,
            "device_name": 120,
            "model": 100,
            "status": 90,
            "rssi": 90,
            "free_heap": 120,
            "uptime": 90,
            "timestamp": 120,
        }

        for col, heading in headers.items():
            self.tree.heading(col, text=heading)
            self.tree.column(col, width=col_widths[col], anchor="center")

        # Barra de rolagem vertical
        scrollbar = ttk.Scrollbar(table_frame, orient=tk.VERTICAL, command=self.tree.yview)
        self.tree.configure(yscroll=scrollbar.set)

        self.tree.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        scrollbar.pack(side=tk.RIGHT, fill=tk.Y)

        # Barra de rodapé
        self.footer = tk.Label(
            root,
            text="Aguardando dados dos sensores...",
            font=("Arial", 9, "italic"),
            fg="gray",
            pady=5,
        )
        self.footer.pack(side=tk.BOTTOM, fill=tk.X)

    def update_data(self, data: dict, topic: str = ""):
        # A identidade vem do SEGMENTO DO TOPICO (devices/{MAC}/health-check),
        # nao do payload. O topico existe tanto no firmware novo (device_id)
        # quanto no antigo (sensor_id); ler do payload faria todas as placas
        # colapsarem na linha "DESCONHECIDO" no dia do rename, porque o iid da
        # tabela e essa chave.
        parts = topic.split("/")
        device_id = parts[1].upper() if len(parts) == 3 and parts[1] else ""
        if not device_id:
            device_id = str(
                data.get("device_id") or data.get("sensor_id") or "DESCONHECIDO"
            )

        # Nome amigavel publicado pelo firmware; sem ele, mostra o proprio MAC.
        device_name = str(data.get("device_name") or device_id)

        model = data.get("sensor_model", "N/A")
        status = str(data.get("status", "N/A")).upper()
        rssi = f"{data.get('rssi', '--')} dBm"

        free_heap_raw = data.get("free_heap", 0)
        free_heap = f"{free_heap_raw / 1024.0:.1f} KB"

        uptime_raw = data.get("uptime_ms", 0)
        uptime = f"{uptime_raw / 1000.0:.1f} s"

        timestamp = str(data.get("timestamp", "--"))

        values = (device_id, device_name, model, status, rssi, free_heap, uptime, timestamp)

        # Atualiza a linha existente ou insere um novo dispositivo
        if self.tree.exists(device_id):
            self.tree.item(device_id, values=values)
        else:
            self.tree.insert("", tk.END, iid=device_id, values=values)

        total = len(self.tree.get_children())
        self.footer.config(
            text=f"Dispositivos ativos: {total} | Última atualização: {device_name}",
            fg="#27ae60",
        )


def start_mqtt(app):
    def on_connect(client, userdata, flags, rc, properties):
        if rc == 0:
            client.subscribe(TOPIC)
            app.footer.config(text=f"Conectado ao broker! Tópico: {TOPIC}")

    def on_message(client, userdata, msg):
        try:
            data = json.loads(msg.payload.decode("utf-8"))
            app.root.after(0, app.update_data, data, msg.topic)
        except Exception:
            pass

    client = mqtt.Client(CallbackAPIVersion.VERSION2, client_id=CLIENT_ID)
    client.on_connect = on_connect
    client.on_message = on_message
    client.connect(BROKER, PORT, keepalive=60)
    client.loop_start()


def main():
    print(f"[monitor] broker tcp://{BROKER}:{PORT} | topico {TOPIC}", flush=True)
    root = tk.Tk()
    app = MultiHealthcheckApp(root)
    start_mqtt(app)
    root.mainloop()


if __name__ == "__main__":
    main()
