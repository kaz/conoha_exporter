package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func indexPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>ConoHa Exporter</title>
		</head>
		<body>
			<h1>ConoHa Exporter</h1>
			<p><a href="/metrics">Metrics</a></p>
		</body>
		</html>
	`))
}

func main() {
	log.Println("ConoHa exporter started.")

	region := flag.String("region", "tyo1", "Region")
	tenantId := flag.String("tenant-id", "", "Your tenant ID")
	username := flag.String("username", "", "Your API user name")
	password := flag.String("password", "", "Your API user password")
	flag.Parse()

	client, err := NewClient(*region, *tenantId, *username, *password)
	if err != nil {
		log.Fatal(err)
	}

	exporter, err := NewConohaCollector(client)
	if err != nil {
		log.Fatal(err)
	}

	if err := prometheus.Register(exporter); err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/", indexPage)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
