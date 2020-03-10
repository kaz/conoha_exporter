package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
)

type Config struct {
	Port     string `yaml:"port"`
	Region   string `yaml:"region"`
	TenantId string `yaml:"tenant_id"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// インデックスページ用 (Prometheusは別にココを触らないので、お好みで……)
func indexPage(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(`
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

	// config.ymlを読み取る
	buf, err := ioutil.ReadFile("./conoha_exporter_config.yaml")
	if err != nil {
		log.Fatal(err)
		return
	}

	var config Config
	if err := yaml.Unmarshal(buf, &config); err != nil {
		log.Fatal(err)
	}

	// ConoHa APIクライアントを作成
	client, err := NewClient(config.Region, config.TenantId, config.Username, config.Password)
	if err != nil {
		log.Fatal(err)
	}

	// 実装したCollectorのを作成
	exporter, err := NewConohaCollector(client)
	if err != nil {
		log.Fatal(err)
	}

	// Collectorをprometheusライブラリに登録
	if err := prometheus.Register(exporter); err != nil {
		log.Fatal(err)
	}

	// 定期的にメトリクスを更新する
	go exporter.AutoUpdate()

	// HTTPでメトリクスを出力
	http.HandleFunc("/", indexPage)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}
