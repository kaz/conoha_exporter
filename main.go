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

	port := flag.String("port", os.Getenv("PORT"), "Port number to listen on")
	region := flag.String("region", "tyo1", "ConoHa region")
	tenantId := flag.String("tenant-id", "", "ConoHa tenant ID")
	username := flag.String("username", "", "ConoHa API user name")
	password := flag.String("password", "", "ConoHa API user password")
	flag.Parse()

	if *port == "" {
		*port = "3000"
	}

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

	go exporter.AutoUpdate()
	http.HandleFunc("/", indexPage)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
