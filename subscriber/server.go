package main

import (
	"os"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)


var influxClient influxdb2.Client
var influxWriteAPI = influxClient.WriteAPIBlocking("", "")



const (
	influxOrg    = "esp32"
	influxBucket = "sensors"
)

func InitInfluxDB() {
	url := os.Getenv("INFLUX_HOST")
	token := os.Getenv("TOKEN_INFLUX")
	org := os.Getenv("INFLUX_ORG")
	bucket := os.Getenv("INFLUX_BUCKET")

	influxClient = influxdb2.NewClient(url, token)

	influxWriteAPI = influxClient.WriteAPIBlocking(
		org,
		bucket,
	)


}


func CloseInfluxDB() {
	influxClient.Close()
}

//Irei escrever usando o writeBlocking, para deixar mais escalavel precisa usar o writepoint async, testando eficiencia