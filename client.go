package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type ConohaClient struct {
	http.Client
	region                  string
	requestNewToken         func(c *http.Client) (*TokenResponse, error)
	token                   string
	tokenExpires            time.Time
	accountEndpoint         string
	databaseHostingEndpoint string
}

// JSON 受け取り用
type TokenResponse struct {
	Access Access `json:"access"`
}
type Token struct {
	IssuedAt string      `json:"issued_at"`
	Expires  time.Time   `json:"expires"`
	ID       string      `json:"id"`
	Tenant   interface{} `json:"tenant"`
	AuditIds []string    `json:"audit_ids"`
}
type Endpoint struct {
	Region    string `json:"region"`
	PublicURL string `json:"publicURL"`
}
type ServiceCatalog struct {
	Endpoints      []Endpoint    `json:"endpoints"`
	EndpointsLinks []interface{} `json:"endpoints_links"`
	Type           string        `json:"type"`
	Name           string        `json:"name"`
}
type Access struct {
	Token          Token            `json:"token"`
	ServiceCatalog []ServiceCatalog `json:"serviceCatalog"`
	User           interface{}      `json:"user"`
	Metadata       interface{}      `json:"metadata"`
}

func NewClient(region string, tenantId string, username string, password string) (*ConohaClient, error) {
	client := &ConohaClient{region: region}

	client.requestNewToken = tokenRequester(region, tenantId, username, password)

	respData, err := client.requestNewToken(&client.Client)
	if err != nil {
		return nil, err
	}

	// 値にアクセス
	access := respData.Access
	serviceCatalog := access.ServiceCatalog

	// トークンを取得
	client.token = access.Token.ID
	client.tokenExpires = access.Token.Expires

	// Account API / Database Hosting APIのエンドポイントを取得
	for _, service := range serviceCatalog {
		switch service.Type {
		case "account":
			client.accountEndpoint = service.Endpoints[0].PublicURL
		case "databasehosting":
			client.databaseHostingEndpoint = service.Endpoints[0].PublicURL
		}
	}

	return client, nil
}

func tokenRequester(region string, tenantId string, username string, password string) func(c *http.Client) (*TokenResponse, error) {
	return func(c *http.Client) (*TokenResponse, error) {
		// リクエストJSONを組み立てる
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

		// トークン発行リクエスト
		resp, err := c.Post("https://identity."+region+".conoha.io/v2.0/tokens", "application/json", bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		// レスポンスボディを取得
		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// JSONを読む
		var respData TokenResponse
		if err := json.Unmarshal(respBytes, &respData); err != nil {
			return nil, err
		}

		return &respData, nil
	}
}

func (cc *ConohaClient) get(url string) ([]byte, error) {
	// トークンの有効性を確認
	if cc.tokenExpires.Before(time.Now().Add(time.Minute)) {
		log.Println("Renewing token...")
		tokenResp, err := cc.requestNewToken(&cc.Client)
		if err != nil {
			return nil, err
		}
		cc.token = tokenResp.Access.Token.ID
		cc.tokenExpires = tokenResp.Access.Token.Expires
		log.Printf("Renewed token, new expiration date: %v", cc.tokenExpires)
	}

	// GETリクエストを飛ばす
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// ヘッダーにトークンをセットする
	req.Header.Set("X-Auth-Token", cc.token)
	resp, err := cc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスボディを返す
	return ioutil.ReadAll(resp.Body)
}

// JSON 受け取り用
type ObjectStorageRequestsResponse struct {
	Request Usage `json:"request"`
}
type Usage struct {
	Schema []string    `json:"schema"`
	Data   [][]float64 `json:"data"`
}

// オブジェクトストレージへのリクエスト数を取得
func (cc *ConohaClient) ObjectStorageRequests() (map[string]float64, error) {
	// メトリクス取得
	resp, err := cc.get(cc.accountEndpoint + "/object-storage/rrd/request")
	if err != nil {
		return nil, err
	}

	// JSONを読む
	var uResp ObjectStorageRequestsResponse
	if err := json.Unmarshal(resp, &uResp); err != nil {
		return nil, err
	}

	// データ整形
	data := uResp.Request.Data[len(uResp.Request.Data)-3]
	usage := make(map[string]float64)
	for i, label := range uResp.Request.Schema {
		usage[label] = data[i]
	}

	return usage, nil
}

// JSON 受け取り用
type ObjectStorageSizeResponse struct {
	Size Usage `json:"size"`
}

// オブジェクトストレージの使用容量を取得
func (cc *ConohaClient) ObjectStorageUsage() (map[string]float64, error) {
	// メトリクス取得
	resp, err := cc.get(cc.accountEndpoint + "/object-storage/rrd/size")
	if err != nil {
		return nil, err
	}

	// JSONを読む
	var uResp ObjectStorageSizeResponse
	if err := json.Unmarshal(resp, &uResp); err != nil {
		return nil, err
	}

	// データ整形
	data := uResp.Size.Data[len(uResp.Size.Data)-3]
	usage := make(map[string]float64)
	for i, label := range uResp.Size.Schema {
		usage[label] = data[i]
	}

	return usage, nil
}

// JSON 受け取り用
type DatabaseListResponse struct {
	TotalCount   int        `json:"total_count"`
	CurrentCount int        `json:"current_count"`
	Databases    []Database `json:"databases"`
}
type Database struct {
	Status           string  `json:"status"`
	InternalHostname string  `json:"internal_hostname"`
	Memo             string  `json:"memo"`
	Charset          string  `json:"charset"`
	DatabaseID       string  `json:"database_id"`
	DbName           string  `json:"db_name"`
	DbSize           float64 `json:"db_size"`
	ServiceID        string  `json:"service_id"`
	ExternalHostname string  `json:"external_hostname"`
	Type             string  `json:"type"`
}

// データベース一覧取得
func (cc *ConohaClient) Databases() ([]*Database, error) {
	resp, err := cc.get(cc.databaseHostingEndpoint + "/databases")
	if err != nil {
		return nil, err
	}

	// JSONを読む
	var uResp DatabaseListResponse
	if err := json.Unmarshal(resp, &uResp); err != nil {
		return nil, err
	}

	databases := make([]*Database, 0)

	for _, d := range uResp.Databases {
		database := d
		databases = append(databases, &database)
	}

	return databases, nil
}

// JSON 受け取り用
type DatabaseQuotaResponse struct {
	Quota Quota `json:"quota"`
}
type Quota struct {
	TotalUsage float64 `json:"total_usage"`
	Quota      int     `json:"quota"`
}

// データベース上限値取得（GB単位）
func (cc *ConohaClient) DatabaseQuota(serviceID string) (*Quota, error) {
	resp, err := cc.get(cc.databaseHostingEndpoint + "/services/" + serviceID + "/quotas")
	if err != nil {
		return nil, err
	}

	// JSONを読む
	var uResp DatabaseQuotaResponse
	if err := json.Unmarshal(resp, &uResp); err != nil {
		return nil, err
	}

	return &uResp.Quota, nil
}

// JSON 受け取り用
type DatabaseInfoResponse struct {
	Database Database `json:"database"`
}

// データベース情報取得（GB単位）
func (cc *ConohaClient) DatabaseInfo(databaseID string) (*Database, error) {
	resp, err := cc.get(cc.databaseHostingEndpoint + "/databases/" + databaseID)
	if err != nil {
		return nil, err
	}

	// JSONを読む
	var uResp DatabaseInfoResponse
	if err := json.Unmarshal(resp, &uResp); err != nil {
		return nil, err
	}

	return &uResp.Database, nil
}
