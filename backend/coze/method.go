package coze

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RequestBot 发送聊天请求
func (c *CozeClient) RequestBot(content string) (ApiResponse, error) {
	requestBody := map[string]interface{}{
		"bot_id":            c.botID,
		"user_id":           "123321",
		"stream":            false,
		"auto_save_history": true,
		"additional_messages": []map[string]interface{}{
			{
				"role":         "user",
				"content":      content,
				"content_type": "text",
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return ApiResponse{}, fmt.Errorf("create request: %w", err)
	}

	return c.doRequest(req)
}

// GetResponse 获取单个响应
func (c *CozeClient) GetResponse(conversationID, chatID string) (ApiResponse, error) {
	reqURL := fmt.Sprintf("%s?conversation_id=%s&chat_id=%s", c.retrieveURL, conversationID, chatID)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("create request: %w", err)
	}

	return c.doRequest(req)
}

// GetConversationList 获取会话列表
func (c *CozeClient) GetConversationList(conversationID, chatID string) (ConvResp, error) {
	reqURL := fmt.Sprintf("%s?conversation_id=%s&chat_id=%s", c.listURL, conversationID, chatID)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return ConvResp{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.doRequestRaw(req)
	if err != nil {
		return ConvResp{}, err
	}

	var convRespList ConvResp
	if err := json.Unmarshal(resp, &convRespList); err != nil {
		return ConvResp{}, fmt.Errorf("unmarshal response: %w", err)
	}

	return convRespList, nil
}

// doRequest 处理通用请求并返回 ApiResponse
func (c *CozeClient) doRequest(req *http.Request) (ApiResponse, error) {
	resp, err := c.doRequestRaw(req)
	if err != nil {
		return ApiResponse{}, err
	}

	var apiResp ApiResponse
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return ApiResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	return apiResp, nil
}

// doRequestRaw 处理原始请求
func (c *CozeClient) doRequestRaw(req *http.Request) ([]byte, error) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return body, nil
}
