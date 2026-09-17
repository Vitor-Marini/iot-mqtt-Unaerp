#!/usr/bin/env bash
# ==============================================================================
# Script Simples de Disparo OTA - ESP32
# ==============================================================================

SERVER="${OTA_SERVER_URL:-http://localhost:8080}"
TARGET="all"
VERSION="1.0.1"

# Processa argumentos (posicionais ou com flags)
while [[ $# -gt 0 ]]; do
  case $1 in
    status)
      echo "=== Status Atual dos Nós ==="
      curl --max-time 3 -s "$SERVER/status" | jq . 2>/dev/null || curl --max-time 3 -s "$SERVER/status"
      echo ""
      exit 0
      ;;
    -v|--version)
      VERSION="$2"; shift 2 ;;
    -t|--target)
      TARGET="$2"; shift 2 ;;
    -s|--server)
      SERVER="$2"; shift 2 ;;
    -h|--help)
      echo "Uso:"
      echo "  $0                  (Dispara v1.0.1 para todas as placas)"
      echo "  $0 1.0.5            (Dispara v1.0.5 para todas as placas)"
      echo "  $0 1.0.5 <MAC>      (Dispara v1.0.5 para um nó específico)"
      echo "  $0 status           (Consulta o status das placas)"
      exit 0
      ;;
    *)
      if [ -z "$POS1" ]; then
        VERSION="$1"
        POS1=1
      elif [ -z "$POS2" ]; then
        TARGET="$1"
        POS2=1
      fi
      shift ;;
  esac
done

echo "=========================================="
echo "    DISPARO DE ATUALIZAÇÃO OTA (ESP32)"
echo "=========================================="
echo " Servidor : $SERVER"
echo " Versão   : $VERSION"
echo " Alvo     : $TARGET"
echo "=========================================="

echo -n "🚀 Enviando comando via MQTT... "

RESPONSE=$(curl --max-time 5 -s -X POST "$SERVER/api/ota/publish" \
  -H "Content-Type: application/json" \
  -d "{\"target\": \"$TARGET\", \"version\": \"$VERSION\"}" 2>&1)

if [ $? -ne 0 ]; then
  echo "FALHOU!"
  echo "Erro de conexão com o servidor OTA ($SERVER):"
  echo "$RESPONSE"
  echo "Verifique se o servidor Go está rodando em outro terminal com 'make run'!"
  exit 1
fi

echo "OK!"
echo "Resposta do servidor:"
echo "$RESPONSE"
echo ""
echo "-> Acompanhe o progresso em tempo real nos logs do servidor ('make run')!"
