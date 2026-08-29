package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer() {

	http.Handle(
		"/metrics",
		promhttp.Handler(),
	)

	fmt.Println("Metrics: http://localhost:8080/metrics")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}