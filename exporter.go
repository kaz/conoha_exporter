package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ConohaCollector struct {
	*ConohaClient
	metrics []*prometheus.Desc
	servers []Server
}

func NewConohaCollector(client *ConohaClient) (*ConohaCollector, error) {
	servers, err := client.Servers()
	if err != nil {
		return nil, err
	}

	return &ConohaCollector{
		client,
		[]*prometheus.Desc{
			prometheus.NewDesc("conoha_cpu", "CPU usage of ConoHa instance", []string{"instance"}, nil),
			prometheus.NewDesc("conoha_disk", "Disk usage of ConoHa instance", []string{"instance", "rw"}, nil),
			prometheus.NewDesc("conoha_interface", "Interface usage of ConoHa instance", []string{"instance", "mac", "direction"}, nil),
		},
		servers,
	}, nil
}

func (cc *ConohaCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range cc.metrics {
		ch <- desc
	}
}
func (cc *ConohaCollector) Collect(ch chan<- prometheus.Metric) {
	for _, srv := range cc.servers {
		cpu, err := cc.CpuUsage(srv)
		if err != nil {
			panic(err)
		}
		ch <- prometheus.MustNewConstMetric(cc.metrics[0], prometheus.GaugeValue, cpu["value"], srv.Name)

		disk, err := cc.DiskUsage(srv)
		if err != nil {
			panic(err)
		}
		ch <- prometheus.MustNewConstMetric(cc.metrics[1], prometheus.GaugeValue, disk["read"], srv.Name, "read")
		ch <- prometheus.MustNewConstMetric(cc.metrics[1], prometheus.GaugeValue, disk["write"], srv.Name, "write")

		for _, ifaceDef := range srv.Interfaces {
			iface, err := cc.InterfaceUsage(srv, ifaceDef)
			if err != nil {
				panic(err)
			}
			ch <- prometheus.MustNewConstMetric(cc.metrics[2], prometheus.GaugeValue, iface["rx"], srv.Name, ifaceDef.MacAddr, "rx")
			ch <- prometheus.MustNewConstMetric(cc.metrics[2], prometheus.GaugeValue, iface["tx"], srv.Name, ifaceDef.MacAddr, "tx")
		}
	}
}
