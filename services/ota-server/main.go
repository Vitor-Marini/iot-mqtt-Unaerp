package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

// getPrimaryLANIP localiza a interface de rede física (Wi-Fi ou Ethernet) e retorna o IP local
func getPrimaryLANIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	// 1. Procura primeiro interfaces físicas Wi-Fi ou Ethernet (wl*, en*, eth*)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(iface.Name)
		if strings.HasPrefix(name, "wl") || strings.HasPrefix(name, "en") || strings.HasPrefix(name, "eth") {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
					return ip.String()
				}
			}
		}
	}

	// 2. Se não encontrou por prefixo físico, pega a primeira interface UP que não seja docker/vpn
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(iface.Name)
		if strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "tailscale") || strings.HasPrefix(name, "wg") ||
			strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "veth") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}

	return "127.0.0.1"
}

func getLocalIPv4s() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil {
				ips = append(ips, fmt.Sprintf("%s (%s)", ip.String(), iface.Name))
			}
		}
	}
	return ips
}

func getEnvWithSource(key, defaultVal string) (string, string) {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val, ".env / ambiente"
	}
	return defaultVal, "padrão fallback"
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func main() {
	// Carrega .env procurando no diretório atual ou nos caminhos comuns do monorepo
	var loadedFile string
	for _, p := range []string{".env", "services/ota-server/.env", "../../.env"} {
		if err := godotenv.Load(p); err == nil {
			loadedFile = p
			break
		}
	}

	otaPort := getEnvInt("OTA_PORT", 8080)
	externalHost, extSrc := getEnvWithSource("OTA_EXTERNAL_HOST", "auto")
	if externalHost == "auto" || externalHost == "" {
		detected := getPrimaryLANIP()
		if detected != "127.0.0.1" {
			externalHost = detected
			extSrc = "auto-detectado da placa de rede"
		}
	}

	// Permite sobrescrever via argumento: go run . 192.168.10.130 ou make run HOST=192.168.10.130
	if len(os.Args) > 1 && os.Args[1] != "" && !strings.HasPrefix(os.Args[1], "-") {
		externalHost = os.Args[1]
		extSrc = "argumento de linha de comando"
	} else if val, ok := os.LookupEnv("HOST"); ok && val != "" {
		externalHost = val
		extSrc = "variavel de ambiente HOST"
	}

	storageDir, _ := getEnvWithSource("STORAGE_DIR", "./storage")
	mqttHost, mqttSrc := getEnvWithSource("MQTT_HOST", "localhost")
	mqttPort := getEnvInt("MQTT_PORT", 1883)
	mqttUser, _ := getEnvWithSource("MQTT_USER", "")
	mqttPass, _ := getEnvWithSource("MQTT_PASS", "")
	mqttTopicBase, _ := getEnvWithSource("MQTT_TOPIC_BASE", "devices")

	_ = os.MkdirAll(storageDir, 0755)

	log.Println("==================================================")
	log.Println("            OTA FIRMWARE SERVER (ESP32)           ")
	log.Println("==================================================")
	if loadedFile != "" {
		log.Printf(" Arquivo .env     : Lido com sucesso de '%s'", loadedFile)
	} else {
		log.Println(" Arquivo .env     : Nenhum .env encontrado (usando variáveis de ambiente ou padrão)")
	}
	log.Printf(" Porta HTTP       : %d", otaPort)
	log.Printf(" Host Externo     : %s [%s]", externalHost, extSrc)
	log.Printf(" Broker MQTT      : %s:%d [%s]", mqttHost, mqttPort, mqttSrc)
	log.Printf(" Pasta de Storage : %s", storageDir)

	localIPs := getLocalIPv4s()
	if len(localIPs) > 0 {
		log.Println(" IPs locais detectados nesta máquina:")
		for _, ipStr := range localIPs {
			log.Printf("  -> %s", ipStr)
		}
	}
	log.Println("==================================================")

	// Inicializa cliente MQTT
	mqttMgr, err := NewMQTTManager(mqttHost, mqttPort, mqttUser, mqttPass, mqttTopicBase)
	if err != nil {
		log.Fatalf("[FATAL] Erro ao configurar MQTT: %v", err)
	}

	// Inicializa servidor HTTP
	httpService := NewHTTPServer(storageDir, externalHost, otaPort, mqttMgr)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", otaPort),
		Handler: httpService.SetupRoutes(),
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[HTTP] Servidor escutando em http://0.0.0.0:%d", otaPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Falha no servidor HTTP: %v", err)
		}
	}()

	<-stopChan
	log.Println("\n[SHUTDOWN] Desligando servidor OTA...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	log.Println("[SHUTDOWN] Finalizado.")
}
