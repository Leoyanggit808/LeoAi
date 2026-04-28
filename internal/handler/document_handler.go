package handler

import (
	"LeoAi/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	documentService *service.DocumentService
}

func NewDocumentHandler(documentService *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{documentService: documentService}
}

// 上传文档
func (h *DocumentHandler) Upload(c *gin.Context) {
	userID := c.GetUint("user_id") // 从 JWT 中间件获取

	title := c.PostForm("title") //获取表单字段 title
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文档标题不能为空"})
		return
	}

	file, header, err := c.Request.FormFile("file") //c.Request.FormFile是 Gin 中处理文件上传的核心方法。
	//FormFile("file") 中的 "file" 是前端表单中传来的 input 标签的 name 属性。
	//file  文件内容流，可以用来读取文件数据
	//header 文件元信息（文件名、大小、MIME 类型等） 是一个*multipart.FileHeader类型对象 有一些字段，如：header.Filename（原始文件名，例如 "报告.pdf"） header.Size header.Header
	//err 错误信息（没有文件、读取失败等）
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}
	defer file.Close()

	doc, err := h.documentService.UploadDocument(userID, title, file, header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "文档上传成功",
		"document": doc, //调用service层中的UploadDocument方法返回的doc 即我们保存到数据库中的文档
	})
}

// 获取用户文档列表
func (h *DocumentHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")

	docs, err := h.documentService.GetUserDocuments(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"documents": docs})
}
