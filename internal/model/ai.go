package model

// 请求ai
type GenerateRequest struct {
	Topic string `json:"topic" binding:"required"`
}

// ai回复
type GenerateResponse struct {
	Content string `json:"content"`
}
