package model

import "time"

type Document struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Title     string    `gorm:"not null" json:"title"`
	FilePath  string    `json:"file_path"` // 文件实际存储路径 保存上传的文件在服务器上的路径
	Content   string    `gorm:"type:text" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type AITask struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index" json:"user_id"`
	DocumentID  uint       `json:"document_id,omitempty"`             //omitempty 当该字段的值为“空值”时，在序列化为 JSON 时忽略该字段（不输出到 JSON 字符串中）。
	TaskType    string     `gorm:"not null" json:"task_type"`         // generate, improve, rag
	Input       string     `gorm:"type:text" json:"input"`            //输入内容
	Output      string     `gorm:"type:text" json:"output,omitempty"` //生成结果
	Status      string     `gorm:"default:'pending'" json:"status"`   // pending, processing, completed, failed
	ErrorMsg    string     `json:"error_msg,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
