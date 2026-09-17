package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// OTACommandPayload define o JSON enviado para o ESP32 iniciar a atualização
type OTACommandPayload struct {
	Cmd     string `json:"cmd"`
	Version string `json:"version"`
	URL     string `json:"url"`
	MD5     string `json:"md5,omitempty"`
}

// OTAStatusPayload define o JSON reportado pelo ESP32 sobre o progresso
type OTAStatusPayload struct {
	// DeviceID e informativo: a identidade real vem do segmento do topico.
	DeviceID string `json:"device_id"`
	// SensorID e a chave antiga, de firmware anterior ao rename.
	SensorID  string `json:"sensor_id"`
	Status    string `json:"status"` // "DOWNLOADING", "FLASHING", "SUCCESS", "FAILED"
	Version   string `json:"version"`
	IP        string `json:"ip,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	FreeHeap  uint32 `json:"free_heap,omitempty"`
	UptimeMs  uint64 `json:"uptime_ms,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// NodeTracking guarda o último status conhecido de cada ESP32
type NodeTracking struct {
	LastStatus string    `json:"last_status"`
	Version    string    `json:"version"`
	IP         string    `json:"ip"`
	LastSeen   time.Time `json:"last_seen"`
	Message    string    `json:"message"`
	LastError  string    `json:"last_error,omitempty"`
}

type MQTTManager struct {
	client     mqtt.Client
	topicBase  string
	nodesLock  sync.RWMutex
	nodesState map[string]*NodeTracking
}

func NewMQTTManager(brokerHost string, brokerPort int, user, pass, topicBase string) (*MQTTManager, error) {
	brokerURI := fmt.Sprintf("tcp://%s:%d", brokerHost, brokerPort)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURI)
	opts.SetClientID(fmt.Sprintf("ota-server-%d", time.Now().UnixNano()%100000))
	if user != "" {
		opts.SetUsername(user)
		opts.SetPassword(pass)
	}
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(3 * time.Second)

	mgr := &MQTTManager{
		topicBase:  topicBase,
		nodesState: make(map[string]*NodeTracking),
	}

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Printf("[MQTT] ✅ Conectado ao broker Mosquitto em %s", brokerURI)
		// Assina tópicos de status OTA de todos os nós: devices/+/ota-status de forma assíncrona
		statusTopic := fmt.Sprintf("%s/+/ota-status", topicBase)
		c.Subscribe(statusTopic, 0, mgr.handleOTAStatus)
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("[MQTT] ⚠️ Conexão perdida com o broker: %v. Reconectando...", err)
	})

	client := mqtt.NewClient(opts)
	mgr.client = client

	// Conecta em background para não travar inicialização do HTTP
	go func() {
		token := client.Connect()
		if !token.WaitTimeout(3*time.Second) || token.Error() != nil {
			log.Printf("[MQTT] ⚠️ Aviso: Broker MQTT (%s) não conectado ainda. Tentando em segundo plano...", brokerURI)
		}
	}()

	return mgr, nil
}

// handleOTAStatus processa as mensagens recebidas de cada ESP32
func (m *MQTTManager) handleOTAStatus(client mqtt.Client, msg mqtt.Message) {
	var payload OTAStatusPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		return
	}

	// A identidade vem do TOPICO, nao do payload: foi por ele que o broker
	// roteou a mensagem, e ele existe tanto no firmware novo (device_id) quanto
	// no antigo (sensor_id). Confiar no payload deixaria uma placa com build
	// errado reportar o status de outra.
	parts := strings.Split(msg.Topic(), "/")
	if len(parts) != 3 {
		return
	}
	mac := strings.ToUpper(parts[1])
	if mac == "" {
		return
	}

	m.nodesLock.Lock()
	node, exists := m.nodesState[mac]
	if !exists {
		node = &NodeTracking{}
		m.nodesState[mac] = node
	}
	node.LastStatus = payload.Status
	node.Version = payload.Version
	if payload.IP != "" {
		node.IP = payload.IP
	}
	node.LastSeen = time.Now()
	node.Message = payload.Message
	node.LastError = payload.Error
	m.nodesLock.Unlock()

	ipStr := payload.IP
	if ipStr == "" {
		ipStr = "ip?"
	}

	switch payload.Status {
	case "DOWNLOADING":
		log.Printf("[OTA] ⏳ ESP32 [%s] (%s): Baixando firmware v%s...", mac, ipStr, payload.Version)
	case "FLASHING":
		log.Printf("[OTA] 💾 ESP32 [%s] (%s): Gravando na Flash...", mac, ipStr)
	case "SUCCESS":
		log.Printf("[OTA] ✅ ESP32 [%s] (%s): SUCESSO! Firmware v%s instalado com exito!", mac, ipStr, payload.Version)
	case "FAILED":
		log.Printf("[OTA] ❌ ESP32 [%s] (%s): FALHOU: %s (%s)", mac, ipStr, payload.Error, payload.Message)
	default:
		log.Printf("[OTA] ℹ️ ESP32 [%s] (%s): %s - %s", mac, ipStr, payload.Status, payload.Message)
	}
}

// PublishOTACommand publica o comando de atualizacao com timeout de 2 segundos (nunca trava)
func (m *MQTTManager) PublishOTACommand(target, version, url, md5Hash string) error {
	if m.client == nil || !m.client.IsConnected() {
		return fmt.Errorf("cliente desconectado do broker Mosquitto")
	}

	cmd := OTACommandPayload{
		Cmd:     "ota",
		Version: version,
		URL:     url,
		MD5:     md5Hash,
	}

	payloadBytes, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	var topic string
	if target == "all" || target == "" {
		topic = fmt.Sprintf("%s/broadcast", m.topicBase)
		log.Printf("[OTA] 📢 Enviando comando OTA BROADCAST (v%s) para todas as placas em '%s'...", version, topic)
	} else {
		topic = fmt.Sprintf("%s/%s/commands", m.topicBase, target)
		log.Printf("[OTA] 🎯 Enviando comando OTA UNICAST (v%s) para placa [%s] em '%s'...", version, target, topic)
	}

	token := m.client.Publish(topic, 0, false, payloadBytes)
	if !token.WaitTimeout(2 * time.Second) {
		return fmt.Errorf("timeout de 2s aguardando confirmacao do broker MQTT")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("falha ao publicar no MQTT: %w", err)
	}

	log.Printf("[OTA] 🚀 Comando publicado com sucesso! URL: %s", url)
	return nil
}

// GetNodesStatus retorna o mapa atualizado de estados de todos os nós
func (m *MQTTManager) GetNodesStatus() map[string]NodeTracking {
	m.nodesLock.RLock()
	defer m.nodesLock.RUnlock()

	result := make(map[string]NodeTracking, len(m.nodesState))
	for k, v := range m.nodesState {
		result[k] = *v
	}
	return result
}
