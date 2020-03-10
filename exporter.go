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
	databases []*Database
}

func NewConohaCollector(client *ConohaClient) (*ConohaCollector, error) {
	// データベース一覧を取得
	databases, err := client.Databases()
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
			prometheus.NewDesc("object_storage_requests", "Requests to Object Storage", []string{"method"}, nil),
			prometheus.NewDesc("object_storage_usage", "Usage of Object Storage", []string{}, nil),
			prometheus.NewDesc("database_usage", "Usage of Database (GB)", []string{"database"}, nil),
		},
		[]prometheus.Metric{},
		databases,
	}, nil
}

func (cc *ConohaCollector) AutoUpdate() {
	for {
		metrics := make([]prometheus.Metric, 0)

		// オブジェクトストレージへのリクエスト数を取得
		requests, err := cc.ObjectStorageRequests()
		if err != nil {
			log.Fatal(err)
		}
		metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[0], prometheus.GaugeValue, requests["get"], "get"))
		metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[0], prometheus.GaugeValue, requests["put"], "put"))
		metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[0], prometheus.GaugeValue, requests["delete"], "delete"))

		// オブジェクトストレージ使用容量を取得
		usage, err := cc.ObjectStorageUsage()
		metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[1], prometheus.GaugeValue, usage["value"]))

		serviceIDs := make(map[string]bool)

		for _, db := range cc.databases {
			// データベース使用状況を取得
			info, err := cc.DatabaseInfo(db.DatabaseID)
			if err != nil {
				log.Fatal(err)
			}
			metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[2], prometheus.GaugeValue, info.DbSize, db.DbName))

			serviceIDs[db.ServiceID] = true
		}

		for serviceID := range serviceIDs {
			// データベース上限値取得
			quota, err := cc.DatabaseQuota(serviceID)
			if err != nil {
				log.Fatal(err)
			}
			metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[2], prometheus.GaugeValue, float64(quota.Quota), "Quota "+serviceID))
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
