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
	servers, err := client.Servers()
	if err != nil {
		return nil, err
	}

	return &ConohaCollector{
		client,
		&sync.RWMutex{},
		[]*prometheus.Desc{
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
			cpu, err := cc.CpuUsage(srv)
			if err != nil {
				log.Fatal(err)
			}
			metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[0], prometheus.GaugeValue, cpu["value"], srv.Name))

			disk, err := cc.DiskUsage(srv)
			if err != nil {
				log.Fatal(err)
			}
			metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[1], prometheus.GaugeValue, disk["read"], srv.Name, "read"))
			metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[1], prometheus.GaugeValue, disk["write"], srv.Name, "write"))

			for _, ifaceDef := range srv.Interfaces {
				iface, err := cc.InterfaceUsage(srv, ifaceDef)
				if err != nil {
					log.Fatal(err)
				}
				metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[2], prometheus.GaugeValue, iface["rx"], srv.Name, ifaceDef.MacAddr, "rx"))
				metrics = append(metrics, prometheus.MustNewConstMetric(cc.describes[2], prometheus.GaugeValue, iface["tx"], srv.Name, ifaceDef.MacAddr, "tx"))
			}
		}

		cc.Lock()
		cc.metrics = metrics
		cc.Unlock()

		log.Println("Metrics updated.")

		<-time.NewTimer(70 * time.Second).C
	}
}

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
