package main

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ConohaCollector struct {
	*ConohaClient
	*sync.RWMutex
	describes []*prometheus.Desc
	metrics   []prometheus.Metric
	servers   []Server
}

func NewConohaCollector(client *ConohaClient) (*ConohaCollector, error) {
	// インスタンス一覧取得
	servers, err := client.Servers()
	if err != nil {
		return nil, err
	}

	// 提供するメトリクスの情報などを定義
	return &ConohaCollector{
		client,
		&sync.RWMutex{},
		[]*prometheus.Desc{
			// NewDescの3番目の引数は可変ラベル（NewConstMetricの最後の可変長引数に対応してる）
			// 4番目のnilには、固定ラベルをprometheus.Labelsで渡せる
			prometheus.NewDesc("conoha_cpu", "CPU usage of ConoHa instance", []string{"instance"}, nil),
			prometheus.NewDesc("conoha_disk", "Disk usage of ConoHa instance", []string{"instance", "rw"}, nil),
			prometheus.NewDesc("conoha_interface", "Interface usage of ConoHa instance", []string{"instance", "mac", "direction"}, nil),
		},
		[]prometheus.Metric{},
		servers,
	}, nil
}

func (cc *ConohaCollector) AutoUpdate() {
	for {
		metrics := []prometheus.Metric{}

		for _, srv := range cc.servers {
			// CPU使用状況を取得
			cpu, err := cc.CpuUsage(srv)
			if err != nil {
				log.Fatal(err)
			}
			metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[0], prometheus.GaugeValue, cpu["value"], srv.Name))

			// ディスク使用状況を取得
			disk, err := cc.DiskUsage(srv)
			if err != nil {
				log.Fatal(err)
			}
			metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[1], prometheus.GaugeValue, disk["read"], srv.Name, "read"))
			metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[1], prometheus.GaugeValue, disk["write"], srv.Name, "write"))

			// インタフェース使用状況を取得
			for _, ifaceDef := range srv.Interfaces {
				iface, err := cc.InterfaceUsage(srv, ifaceDef)
				if err != nil {
					log.Fatal(err)
				}
				metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[2], prometheus.GaugeValue, iface["rx"], srv.Name, ifaceDef.MacAddr, "rx"))
				metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[2], prometheus.GaugeValue, iface["tx"], srv.Name, ifaceDef.MacAddr, "tx"))
			}
		}

		// メトリクスデータ更新
		cc.Lock()
		cc.metrics = metrics
		cc.Unlock()

		log.Println("Metrics updated.")

		// 70秒間待機（ConoHa API側の更新間隔）
		<-time.NewTimer(70 * time.Second).C
	}
}

// 内部で保持しているデータを返す
func (cc *ConohaCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range cc.describes {
		ch <- d
	}
}
func (cc *ConohaCollector) Collect(ch chan<- prometheus.Metric) {
	cc.RLock()
	defer cc.RUnlock()

	for _, m := range cc.metrics {
		ch <- m
	}
}
