package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPServer_Routes(t *testing.T) {
	tempDir := t.TempDir()
	binContent := []byte("ESP32_MOCK_FIRMWARE_BINARY_V1")
	binPath := filepath.Join(tempDir, "firmware-v1.0.0.bin")
	if err := os.WriteFile(binPath, binContent, 0644); err != nil {
		t.Fatalf("falha ao criar arquivo de teste: %v", err)
	}

	server := NewHTTPServer(tempDir, "192.168.1.100", 8080, &MQTTManager{
		topicBase:  "devices",
		nodesState: make(map[string]*NodeTracking),
	})
	handler := server.SetupRoutes()

	t.Run("GET /health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("esperado 200 OK, recebido %d", rec.Code)
		}

		var body map[string]string
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("erro ao decodificar JSON: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("status inesperado: %s", body["status"])
		}
	})

	t.Run("GET /api/firmware/latest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/firmware/latest", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("esperado 200 OK, recebido %d", rec.Code)
		}

		var meta FirmwareMetadata
		if err := json.NewDecoder(rec.Body).Decode(&meta); err != nil {
			t.Fatalf("erro ao decodificar JSON: %v", err)
		}

		if meta.Filename != "firmware-v1.0.0.bin" {
			t.Errorf("esperado firmware-v1.0.0.bin, recebido %s", meta.Filename)
		}
		if meta.Size != int64(len(binContent)) {
			t.Errorf("tamanho esperado %d, recebido %d", len(binContent), meta.Size)
		}
		if meta.MD5 == "" {
			t.Error("MD5 não pode ser vazio")
		}
	})

	t.Run("GET /firmware/firmware-v1.0.0.bin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/firmware/firmware-v1.0.0.bin", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("esperado 200 OK, recebido %d", rec.Code)
		}

		if rec.Header().Get("Content-Type") != "application/octet-stream" {
			t.Errorf("Content-Type inesperado: %s", rec.Header().Get("Content-Type"))
		}
		if rec.Header().Get("x-MD5") == "" {
			t.Error("Header x-MD5 esperado")
		}
		if rec.Body.String() != string(binContent) {
			t.Errorf("conteúdo baixado diverge do original")
		}
	})

	t.Run("GET /api/nodes/status", func(t *testing.T) {
		// Mock de um nó
		server.mqttMgr.nodesState["A1B2C3D4E5F6"] = &NodeTracking{
			LastStatus: "SUCCESS",
			Version:    "1.0.1",
			IP:         "192.168.1.150",
		}

		req := httptest.NewRequest(http.MethodGet, "/api/nodes/status", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("esperado 200 OK, recebido %d", rec.Code)
		}

		var nodes map[string]NodeTracking
		if err := json.NewDecoder(rec.Body).Decode(&nodes); err != nil {
			t.Fatalf("erro ao decodificar JSON: %v", err)
		}

		node, exists := nodes["A1B2C3D4E5F6"]
		if !exists {
			t.Fatal("esperado nó A1B2C3D4E5F6 no status")
		}
		if node.LastStatus != "SUCCESS" || node.Version != "1.0.1" {
			t.Errorf("dados inesperados do nó: %+v", node)
		}
	})

	t.Run("POST /api/ota/publish (sem MQTT conectado)", func(t *testing.T) {
		payload := `{"target": "all", "version": "1.0.1", "filename": "firmware-v1.0.0.bin"}`
		req := httptest.NewRequest(http.MethodPost, "/api/ota/publish", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Deve retornar erro 500 porque o MQTT não está conectado nos testes unitários
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("esperado 500 ao tentar publicar sem MQTT, recebido %d", rec.Code)
		}
	})
}
