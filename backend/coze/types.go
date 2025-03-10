package coze

// ApiResponse 定义API响应结构
type ApiResponse struct {
	Data struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversation_id"`
		BotID          string `json:"bot_id"`
		CreatedAt      int64  `json:"created_at"`
		LastError      struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		} `json:"last_error"`
		Status string `json:"status"`
	} `json:"data"`
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ConvResp 定义会话列表响应结构
type ConvResp struct {
	Data []ConvRespDataItem `json:"data"`
	Code int                `json:"code"`
	Msg  string             `json:"msg"`
}

// ConvRespDataItem 定义会话列表项结构
type ConvRespDataItem struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversation_id"`
	BotID          string                 `json:"bot_id"`
	ChatID         string                 `json:"chat_id"`
	MetaData       map[string]interface{} `json:"meta_data"`
	Role           string                 `json:"role"`
	Content        string                 `json:"content"`
	ContentType    string                 `json:"content_type"`
	CreatedAt      int64                  `json:"created_at"`
	UpdatedAt      int64                  `json:"updated_at"`
	Type           string                 `json:"type"`
}
