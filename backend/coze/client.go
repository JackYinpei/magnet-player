package coze

import (
	"log"
	"net/http"
	"os"
	"strings"
)

// Region 定义支持的区域
type Region string

const (
	RegionCN  Region = "cn"
	RegionCOM Region = "com"
)

// CozeClient 处理与 Coze API 的所有交互
type CozeClient struct {
	region      Region
	token       string
	botID       string
	baseURL     string
	listURL     string
	retrieveURL string
	client      *http.Client
}

// NewCozeClient 创建新的 CozeClient 实例
func NewCozeClient(region Region) *CozeClient {
	var token, botID, baseURL, retrieveURL, listURL string

	switch region {
	case RegionCN:
		token = os.Getenv("COZECNTOKEN")
		botID = os.Getenv("COZECNBOT")
		baseURL = os.Getenv("COZECNURL")
		retrieveURL = os.Getenv("COZECNRETRIEVEURL")
		listURL = os.Getenv("COZECNLISTURL")
	default: // RegionCOM
		token = os.Getenv("COZECOMTOKEN")
		botID = os.Getenv("COZECOMBOT")
		baseURL = os.Getenv("COZECOMURL")
		retrieveURL = os.Getenv("COZECOMRETRIEVEURL")
		listURL = os.Getenv("COZECOMLISTURL")
	}

	// 验证 URL
	if baseURL == "" || retrieveURL == "" || listURL == "" {
		log.Printf("Warning: Invalid URLs - baseURL: %s, retrieveURL: %s, listURL: %s",
			baseURL, retrieveURL, listURL)
	}

	// 确保所有 URL 都是有效的
	baseURL = strings.TrimSpace(baseURL)
	retrieveURL = strings.TrimSpace(retrieveURL)
	listURL = strings.TrimSpace(listURL)

	return &CozeClient{
		region:      region,
		token:       token,
		botID:       botID,
		baseURL:     baseURL,
		retrieveURL: retrieveURL,
		listURL:     listURL,
		client:      &http.Client{},
	}
}
