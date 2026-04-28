package repository

import (
	"LeoAi/internal/model"

	"gorm.io/gorm"
)

type DocumentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) Create(doc *model.Document) error {
	return r.db.Create(doc).Error
}

func (r *DocumentRepository) FindByID(id uint) (*model.Document, error) {
	var doc model.Document
	err := r.db.First(&doc, id).Error
	return &doc, err
}

func (r *DocumentRepository) FindByUserID(userID uint) ([]model.Document, error) {
	var docs []model.Document
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&docs).Error
	return docs, err
}

func (r *DocumentRepository) UpdateContent(id uint, content string) error {
	return r.db.Model(&model.Document{}).
		Where("id = ?", id).
		Update("content", content).Error
}
