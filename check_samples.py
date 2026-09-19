#!/usr/bin/env python3
"""Script rápido para consultar a contagem e amostragem de dados no InfluxDB.

Executa a consulta Flux via container Docker do InfluxDB e exibe um resumo
formatado no terminal com as contagens por estação, medição e campo.
"""

import subprocess
import sys
from datetime import datetime

# Mapeamento de MAC para nome amigável
STATION_NAMES = {
    "545BA26062EC": "Estação 1",
    "688A7CC0E8FC": "Estação 2",
    "F8E843F7C630": "Estação 3",
}

FLUX_QUERY = """
from(bucket: "sensors")
  |> range(start: 0)
  |> group(columns: ["_measurement", "_field", "sensor_id"])
  |> count()
"""


def fetch_counts():
    try:
        cmd = [
            "docker",
            "exec",
            "influxdb",
            "influx",
            "query",
            "--raw",
            "--org",
            "esp32",
            FLUX_QUERY,
        ]
        res = subprocess.check_output(cmd, stderr=subprocess.PIPE).decode("utf-8")
    except subprocess.CalledProcessError as e:
        print(f"Erro ao consultar InfluxDB: {e.stderr.decode('utf-8')}", file=sys.stderr)
        sys.exit(1)
    except FileNotFoundError:
        print("Erro: comando 'docker' nao encontrado.", file=sys.stderr)
        sys.exit(1)

    data = {}
    for line in res.strip().split("\n"):
        if line.startswith("#") or line.startswith(",result") or not line.strip():
            continue
        parts = line.split(",")
        if len(parts) >= 9:
            try:
                count = int(parts[5])
                measurement = parts[6].strip()
                field = parts[7].strip()
                sensor_id = parts[8].strip()
                data.setdefault(measurement, {}).setdefault(sensor_id, {})[field] = count
            except (ValueError, IndexError):
                pass
    return data


def format_table(data):
    now_str = datetime.now().strftime("%d/%m/%Y %H:%M:%S")
    print("=" * 72)
    print(f" AMOSTRAGEM E CONTAGEM DE DADOS - INFLUXDB (sensors / esp32)")
    print(f" Consulta realizada em: {now_str}")
    print("=" * 72)

    # 1. Telemetria
    print("\n--- 1. TELEMETRIA CLIMATICA (Intervalo: ~5s por leitura) ---")
    print(f"{'Estação':<12} {'MAC / Sensor ID':<16} {'Temp':<8} {'Pressão':<9} {'Altitude':<10} {'Leituras':<10}")
    print("-" * 72)

    telemetry_data = data.get("telemetry", {})
    total_telemetry_reads = 0
    total_telemetry_points = 0

    all_macs = sorted(list(set(list(telemetry_data.keys()) + list(data.get("healthcheck", {}).keys()))))

    for mac in all_macs:
        fields = telemetry_data.get(mac, {})
        temp = fields.get("temperature", 0)
        press = fields.get("pressure", 0)
        alt = fields.get("altitude", 0)
        reads = max(temp, press, alt)
        points = temp + press + alt
        total_telemetry_reads += reads
        total_telemetry_points += points
        name = STATION_NAMES.get(mac, "Desconhecido")
        print(f"{name:<12} {mac:<16} {temp:<8} {press:<9} {alt:<10} {reads:<10}")

    print("-" * 72)
    print(f"{'TOTAL TELEMETRIA:':<48} {total_telemetry_reads:<10} ({total_telemetry_points} pontos)")

    # 2. Healthcheck
    print("\n--- 2. DIAGNOSTICO E HEALTHCHECK (Intervalo: ~30s por leitura) ---")
    print(f"{'Estação':<12} {'MAC / Sensor ID':<16} {'Status':<8} {'RSSI':<8} {'Heap':<9} {'Uptime':<8} {'Leituras':<10}")
    print("-" * 72)

    health_data = data.get("healthcheck", {})
    total_health_reads = 0
    total_health_points = 0

    for mac in all_macs:
        fields = health_data.get(mac, {})
        status = fields.get("status", 0)
        rssi = fields.get("rssi", 0)
        heap = fields.get("free_heap", 0)
        uptime = fields.get("uptime_ms", 0)
        reads = max(status, rssi, heap, uptime)
        points = status + rssi + heap + uptime
        total_health_reads += reads
        total_health_points += points
        name = STATION_NAMES.get(mac, "Desconhecido")
        print(f"{name:<12} {mac:<16} {status:<8} {rssi:<8} {heap:<9} {uptime:<8} {reads:<10}")

    print("-" * 72)
    print(f"{'TOTAL HEALTHCHECK:':<53} {total_health_reads:<10} ({total_health_points} pontos)")

    # 3. Resumo Consolidado
    grand_total_reads = total_telemetry_reads + total_health_reads
    grand_total_points = total_telemetry_points + total_health_points

    print("\n" + "=" * 72)
    print(f" RESUMO GERAL:")
    print(f" - Total de leituras (mensagens MQTT processadas): {grand_total_reads:,}")
    print(f" - Total de pontos de series temporais no banco:   {grand_total_points:,}")
    print("=" * 72)


def main():
    data = fetch_counts()
    format_table(data)


if __name__ == "__main__":
    main()
