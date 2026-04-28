package service

import (
	"LeoAi/internal/model"
	"LeoAi/internal/repository"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DocumentService struct {
	repo          *repository.DocumentRepository
	vectorService *VectorService // ← 新增
}

func NewDocumentService(repo *repository.DocumentRepository, vectorService *VectorService) *DocumentService {
	return &DocumentService{
		repo:          repo,
		vectorService: vectorService,
	}
}

// 上传文档（清理版）
func (s *DocumentService) UploadDocument(userID uint, title string, file io.Reader, filename string) (*model.Document, error) {
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, err
	}

	ext := filepath.Ext(filename)
	newFilename := fmt.Sprintf("%d_%d%s", userID, time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, newFilename)

	out, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return nil, err
	}

	doc := &model.Document{
		UserID:   userID,
		Title:    title,
		FilePath: filePath,
	}

	if err := s.repo.Create(doc); err != nil {
		return nil, err
	}

	fmt.Printf("✅ 文档上传成功: %s (ID: %d)\n", title, doc.ID)
	// 异步向量化（不阻塞上传响应）
	go func() {
		fmt.Printf("\n📂 [文件读取] 开始读取文档内容，路径: %s\n", filePath)

		content, err := readFileContent(filePath, filename)
		if err != nil {
			fmt.Printf("⚠️  [文件读取] 文档 %d 读取失败，跳过向量化: %v\n", doc.ID, err)
			return
		}
		fmt.Printf("✅ [文件读取] 文档 %d 读取成功，共 %d 字\n", doc.ID, len([]rune(content)))

		// 更新数据库 content 字段
		if err := s.repo.UpdateContent(doc.ID, content); err != nil {
			fmt.Printf("⚠️  [文件读取] 文档 %d 更新数据库 content 字段失败: %v\n", doc.ID, err)
		} else {
			fmt.Printf("✅ [文件读取] 文档 %d content 字段已同步至数据库\n", doc.ID)
		}

		// 进入向量化流程
		if err := s.vectorService.AddDocument(doc.ID, content); err != nil {
			fmt.Printf("❌ [文档入库] 文档 %d 向量化流程失败: %v\n", doc.ID, err)
		}
	}()
	return doc, nil
}

// readFileContent 根据文件类型提取纯文本
func readFileContent(filePath, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	//filepath.Ext(filename)：从文件名中提取扩展名（如 ".pdf"、".docx"、".txt"）
	//strings.ToLower(...)：转成小写，统一处理（避免 .TXT 和 .txt 被当成不同类型）
	switch ext {
	case ".txt", ".md":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		// PDF、Word 等格式后续可用专门库处理
		// 目前先返回空，不影响上传流程
		return "", fmt.Errorf("暂不支持的文件格式: %s", ext)
	}
}

// 获取用户文档列表
func (s *DocumentService) GetUserDocuments(userID uint) ([]model.Document, error) {
	return s.repo.FindByUserID(userID)
}

// 根据ID获取文档
func (s *DocumentService) GetDocumentByID(id uint) (*model.Document, error) {
	return s.repo.FindByID(id)
}
