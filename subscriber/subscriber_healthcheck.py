import json
import tkinter as tk
from tkinter import ttk
import paho.mqtt.client as mqtt
from paho.mqtt.enums import CallbackAPIVersion

BROKER = "localhost"
PORT = 1883
TOPIC = "/healthcheck"


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
            "sensor_id",
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
            "sensor_id": "Sensor ID",
            "model": "Modelo",
            "status": "Status",
            "rssi": "Sinal (RSSI)",
            "free_heap": "Free Heap",
            "uptime": "Uptime",
            "timestamp": "Timestamp",
        }

        col_widths = {
            "sensor_id": 110,
            "model": 130,
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

    def update_data(self, data: dict):
        sensor_id = str(data.get("sensor_id", "DESCONHECIDO"))
        model = data.get("sensor_model", "N/A")
        status = str(data.get("status", "N/A")).upper()
        rssi = f"{data.get('rssi', '--')} dBm"

        free_heap_raw = data.get("free_heap", 0)
        free_heap = f"{free_heap_raw / 1024.0:.1f} KB"

        uptime_raw = data.get("uptime_ms", 0)
        uptime = f"{uptime_raw / 1000.0:.1f} s"

        timestamp = str(data.get("timestamp", "--"))

        values = (sensor_id, model, status, rssi, free_heap, uptime, timestamp)

        # Atualiza a linha existente ou insere um novo dispositivo
        if self.tree.exists(sensor_id):
            self.tree.item(sensor_id, values=values)
        else:
            self.tree.insert("", tk.END, iid=sensor_id, values=values)

        total = len(self.tree.get_children())
        self.footer.config(
            text=f"Dispositivos ativos: {total} | Última atualização: {sensor_id}",
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
            app.root.after(0, app.update_data, data)
        except Exception:
            pass

    client = mqtt.Client(CallbackAPIVersion.VERSION2)
    client.on_connect = on_connect
    client.on_message = on_message
    client.connect(BROKER, PORT, keepalive=60)
    client.loop_start()


def main():
    root = tk.Tk()
    app = MultiHealthcheckApp(root)
    start_mqtt(app)
    root.mainloop()


if __name__ == "__main__":
    main()