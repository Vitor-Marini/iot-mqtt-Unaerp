package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

type FirmwareMetadata struct {
	Filename    string    `json:"filename"`
	Size        int64     `json:"size_bytes"`
	MD5         string    `json:"md5"`
	ModTime     time.Time `json:"modified_at"`
	DownloadURL string    `json:"download_url"`
}

type PublishRequest struct {
	Target   string `json:"target"`   // "all" ou MAC
	Version  string `json:"version"`  // ex: "1.0.5"
	Filename string `json:"filename"` // opcional
	URL      string `json:"url"`      // opcional
}

type HTTPServer struct {
	storageDir      string
	externalHost    string
	port            int
	mqttMgr         *MQTTManager
	activeDownloads int32
	downloadSem     chan struct{}
}

func NewHTTPServer(storageDir, externalHost string, port int, mqttMgr *MQTTManager) *HTTPServer {
	return &HTTPServer{
		storageDir:   storageDir,
		externalHost: externalHost,
		port:         port,
		mqttMgr:      mqttMgr,
		downloadSem:  make(chan struct{}, 1), // Limita a 1 download por vez: protege USB contra brownout e Wi-Fi contra colisao
	}
}

func (s *HTTPServer) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/firmware/", s.handleDownloadFirmware)
	mux.HandleFunc("/api/firmware/latest", s.handleLatestFirmware)
	mux.HandleFunc("/api/ota/publish", s.handlePublishOTA)
	mux.HandleFunc("/api/ota/active-downloads", s.handleActiveDownloads)
	mux.HandleFunc("/api/nodes/status", s.handleNodesStatus)
	mux.HandleFunc("/status", s.handleNodesStatus)

	return mux
}

func (s *HTTPServer) handleActiveDownloads(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int32{
		"active_downloads": atomic.LoadInt32(&s.activeDownloads),
	})
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "ota-server",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// handleDownloadFirmware entrega o binário em streaming com headers de validação
func (s *HTTPServer) handleDownloadFirmware(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/firmware/")
	if filename == "" || strings.Contains(filename, "..") {
		http.Error(w, "Nome de arquivo invalido", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.storageDir, filename)
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) || fileInfo.IsDir() {
		log.Printf("[HTTP] ❌ Firmware '%s' nao encontrado no storage (%s)", filename, filePath)
		http.Error(w, "Arquivo de firmware nao encontrado", http.StatusNotFound)
		return
	}

	hash, err := calculateMD5(filePath)
	if err != nil {
		http.Error(w, "Erro ao processar binario", http.StatusInternalServerError)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	log.Printf("[HTTP] ⏳ ESP32 (%s) aguardando vez na fila de download de '%s'...", r.RemoteAddr, filename)

	select {
	case s.downloadSem <- struct{}{}:
		defer func() { <-s.downloadSem }()
	case <-r.Context().Done():
		log.Printf("[HTTP] ⚠️ ESP32 (%s) cancelou requisicao antes de iniciar", r.RemoteAddr)
		return
	}

	atomic.AddInt32(&s.activeDownloads, 1)
	defer atomic.AddInt32(&s.activeDownloads, -1)

	log.Printf("[HTTP] 📥 ESP32 (%s) iniciando download de '%s' (%d bytes)...", r.RemoteAddr, filename, fileInfo.Size())

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("x-MD5", hash)

	// Desativa sendfile do kernel Linux usando struct{ io.Reader }{ file }
	// O ESP32 possui buffer TCP (lwIP) muito pequeno (~4KB) e sofre com sendfile kernel-level
	buf := make([]byte, 4096)
	copied, err := io.CopyBuffer(w, struct{ io.Reader }{ file }, buf)
	if err != nil {
		log.Printf("[HTTP] ⚠️ Download interrompido por %s: %v", r.RemoteAddr, err)
		return
	}

	log.Printf("[HTTP] ✅ ESP32 (%s) concluiu o download de '%s' (%d bytes)", r.RemoteAddr, filename, copied)
}

func (s *HTTPServer) handleLatestFirmware(w http.ResponseWriter, r *http.Request) {
	meta, err := s.getLatestFirmwareMetadata()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}

func (s *HTTPServer) handlePublishOTA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo nao permitido. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("JSON invalido: %v", err), http.StatusBadRequest)
		return
	}

	if req.Version == "" {
		req.Version = "1.0.1"
	}
	if req.Target == "" {
		req.Target = "all"
	}

	var targetFile string
	if req.Filename != "" {
		targetFile = filepath.Join(s.storageDir, req.Filename)
	} else {
		meta, err := s.getLatestFirmwareMetadata()
		if err != nil {
			http.Error(w, fmt.Sprintf("Nenhum firmware (.bin) disponivel no storage: %v", err), http.StatusBadRequest)
			return
		}
		targetFile = filepath.Join(s.storageDir, meta.Filename)
		req.Filename = meta.Filename
	}

	md5Hash, err := calculateMD5(targetFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao ler firmware '%s': %v", req.Filename, err), http.StatusInternalServerError)
		return
	}

	downloadURL := req.URL
	if downloadURL == "" {
		downloadURL = fmt.Sprintf("http://%s:%d/firmware/%s", s.externalHost, s.port, req.Filename)
	}

	// Publica no broker MQTT com timeout
	if err := s.mqttMgr.PublishOTACommand(req.Target, req.Version, downloadURL, md5Hash); err != nil {
		log.Printf("[OTA ERROR] %v", err)
		http.Error(w, fmt.Sprintf("Erro ao publicar comando MQTT: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "published",
		"target":       req.Target,
		"version":      req.Version,
		"filename":     req.Filename,
		"url":          downloadURL,
		"md5":          md5Hash,
		"published_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *HTTPServer) handleNodesStatus(w http.ResponseWriter, r *http.Request) {
	statusMap := s.mqttMgr.GetNodesStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusMap)
}

func (s *HTTPServer) getLatestFirmwareMetadata() (*FirmwareMetadata, error) {
	files, err := os.ReadDir(s.storageDir)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler storage: %w", err)
	}

	var newestFile os.FileInfo
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".bin") {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		if newestFile == nil || info.ModTime().After(newestFile.ModTime()) {
			newestFile = info
		}
	}

	if newestFile == nil {
		return nil, fmt.Errorf("nenhum arquivo .bin encontrado na pasta %s", s.storageDir)
	}

	filePath := filepath.Join(s.storageDir, newestFile.Name())
	hash, err := calculateMD5(filePath)
	if err != nil {
		return nil, err
	}

	return &FirmwareMetadata{
		Filename:    newestFile.Name(),
		Size:        newestFile.Size(),
		MD5:         hash,
		ModTime:     newestFile.ModTime(),
		DownloadURL: fmt.Sprintf("http://%s:%d/firmware/%s", s.externalHost, s.port, newestFile.Name()),
	}, nil
}

func calculateMD5(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
