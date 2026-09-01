package main

import (
	"fmt"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"



	"subscriber/models"
)

func main () {

	InitInfluxDB()
	defer CloseInfluxDB()


	//criação dos canais de comunicação das goroutines
	telemetryChan := make(chan models.Telemetry)
	healthcheckChan := make(chan models.Healthcheck)

	go ProcessTelemetry(telemetryChan)
	go ProcessHealthcheck(healthcheckChan)

	//carrega as variaveis do .env
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env não encontrado")
	}

	host := getEnv("MQTT_HOST", "localhost")
	port := getEnv("MQTT_PORT", "1883")


	broker := fmt.Sprintf("tcp://%s:%s", host, port)
	SUBSCRIBER_ID := getEnv("SUBSCRIBER_ID", "subscriber")


	fmt.Println("Criando conexão com MQTT...")
	fmt.Println("Broker:", broker)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(string(SUBSCRIBER_ID))

	opts.OnConnect = func(c mqtt.Client) {
		fmt.Println("Conectando ao Mosquitto")

		fmt.Println("Registrando topicos")
	}


	//função que roda ao subscriber se conectar ao mosquitto
	opts.OnConnect = func(client mqtt.Client) {
		fmt.Println("Conectado ao Mosquitto!")

		topics := map[string]byte{
			"devices/+/telemetry":   0,
			"devices/+/healthcheck": 0,
		}

		//função de subscriber -> realizar a inscrição no topico
		//toda vez que uma mensagem chegar, ele chama a função messsageHandler
		//Quero receber mensagens desses dois tópicos.
		//
		token := client.SubscribeMultiple(
			topics,
			func(client mqtt.Client, msg mqtt.Message) {
				messageHandler(
					client,
					msg,
					telemetryChan,
					healthcheckChan,
				)
			},
		)

		if token.Wait() && token.Error() != nil {
			log.Println("Erro ao assinar tópicos:", token.Error())
			return
		}

		fmt.Println("Inscrito nos tópicos:")
		fmt.Println(" - devices/#/telemetry")
		fmt.Println(" - esp32/healthcheck")
	}


	//FUnção quando o mosquitto for desligado
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Println("Conexão MQTT perdida:", err)
	}

	//cliente mqtt conectado e configurado
	client := mqtt.NewClient(opts)
	
	//Criação da conexão com o mosquitto MQTT

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}


	//bloqueia execução main infinitamente até ctrl - c
	select {}
}



func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

