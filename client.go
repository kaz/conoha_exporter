package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type ConohaClient struct {
	http.Client
	region   string
	token    string
	endpoint string
}

func NewClient(region string, tenantId string, username string, password string) (*ConohaClient, error) {
	client := &ConohaClient{region: region}

	data, err := json.Marshal(map[string]interface{}{
		"auth": map[string]interface{}{
			"passwordCredentials": map[string]string{
				"username": username,
				"password": password,
			},
			"tenantId": tenantId,
		},
	})
	if err != nil {
		return nil, err
	}

	resp, err := client.Post("https://identity."+region+".conoha.io/v2.0/tokens", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	respData := make(map[string]interface{})
	if err := json.Unmarshal(respBytes, &respData); err != nil {
		return nil, err
	}

	access := respData["access"].(map[string]interface{})
	token := access["token"].(map[string]interface{})
	serviceCatalog := access["serviceCatalog"].([]interface{})

	client.token = token["id"].(string)

	for _, service := range serviceCatalog {
		svcMap := service.(map[string]interface{})
		if svcMap["type"].(string) == "compute" {
			client.endpoint = svcMap["endpoints"].([]interface{})[0].(map[string]interface{})["publicURL"].(string)
			break
		}
	}

	return client, nil
}

func (cc *ConohaClient) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", cc.endpoint+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Token", cc.token)
	resp, err := cc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

type ServersResponse struct {
	Servers []Server
}
type Server struct {
	ID         string
	Name       string
	Interfaces []Interface
}

type InterfaceResponse struct {
	InterfaceAttachments []Interface
}
type Interface struct {
	PortID  string `json:"port_id"`
	MacAddr string `json:"mac_addr"`
}

func (cc *ConohaClient) Servers() ([]Server, error) {
	resp, err := cc.get("/servers")
	if err != nil {
		return nil, err
	}

	var sResp ServersResponse
	if err := json.Unmarshal(resp, &sResp); err != nil {
		return nil, err
	}

	servers := []Server{}
	for _, s := range sResp.Servers {
		resp, err := cc.get("/servers/" + s.ID + "/os-interface")
		if err != nil {
			return nil, err
		}

		var iResp InterfaceResponse
		if err := json.Unmarshal(resp, &iResp); err != nil {
			return nil, err
		}

		s.Interfaces = iResp.InterfaceAttachments
		servers = append(servers, s)
	}

	return servers, nil
}

type UsageResponse struct {
	CPU       Usage
	Disk      Usage
	Interface Usage
}
type Usage struct {
	Schema []string
	Data   [][]float64
}

func (cc *ConohaClient) CpuUsage(s Server) (map[string]float64, error) {
	resp, err := cc.get("/servers/" + s.ID + "/rrd/cpu")
	if err != nil {
		return nil, err
	}

	var uResp UsageResponse
	if err := json.Unmarshal(resp, &uResp); err != nil {
		return nil, err
	}

	data := uResp.CPU.Data[len(uResp.CPU.Data)-5]
	usage := make(map[string]float64)
	for i, label := range uResp.CPU.Schema {
		usage[label] = data[i]
	}

	return usage, nil
}
func (cc *ConohaClient) DiskUsage(s Server) (map[string]float64, error) {
	resp, err := cc.get("/servers/" + s.ID + "/rrd/disk")
	if err != nil {
		return nil, err
	}

	var uResp UsageResponse
	if err := json.Unmarshal(resp, &uResp); err != nil {
		return nil, err
	}

	data := uResp.Disk.Data[len(uResp.Disk.Data)-5]
	usage := make(map[string]float64)
	for i, label := range uResp.Disk.Schema {
		usage[label] = data[i]
	}

	return usage, nil
}
func (cc *ConohaClient) InterfaceUsage(s Server, i Interface) (map[string]float64, error) {
	resp, err := cc.get("/servers/" + s.ID + "/rrd/interface?port_id=" + i.PortID)
	if err != nil {
		return nil, err
	}

	var uResp UsageResponse
	if err := json.Unmarshal(resp, &uResp); err != nil {
		return nil, err
	}

	data := uResp.Interface.Data[len(uResp.Interface.Data)-5]
	usage := make(map[string]float64)
	for i, label := range uResp.Interface.Schema {
		usage[label] = data[i]
	}

	return usage, nil
}
