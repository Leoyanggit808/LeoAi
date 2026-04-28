package model

import "time"

type User struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Username         string    `gorm:"unique;not null" json:"username"`
	Email            string    `gorm:"unique;not null" json:"email"`
	PasswordHash     string    `gorm:"not null" json:"-"`
	SubscriptionType string    `gorm:"default:'free'" json:"subscription_type"`
	CreatedAt        time.Time `json:"created_at"`
}
