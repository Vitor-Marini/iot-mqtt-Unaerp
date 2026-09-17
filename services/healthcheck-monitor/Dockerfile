FROM python:3.12-slim

# Nao gera .pyc e nao bufferiza stdout: sem isso os logs do container so
# aparecem em blocos.
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

# A imagem oficial ja traz o modulo _tkinter compilado, mas nao as bibliotecas
# Tcl/Tk que ele carrega em tempo de execucao -- sem elas o `import tkinter`
# falha com "libtk8.6.so: cannot open shared object file".
# `libtk8.6` puxa `libtcl8.6` e e bem menor que o pacote `python3-tk`.
RUN apt-get update \
    && apt-get install -y --no-install-recommends libtk8.6 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY monitor.py ./

RUN useradd --create-home --uid 10001 monitor
USER monitor

# Sem EXPOSE: este servico nao serve nada, so assina MQTT.
# A janela precisa de um servidor X alcancavel via DISPLAY -- ver README.
ENTRYPOINT ["python", "monitor.py"]
