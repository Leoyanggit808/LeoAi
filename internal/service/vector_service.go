package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChromaDB 的 collection 名称，每个用户共用一个
const chromaCollection = "leoai_documents"

type VectorService struct {
	chromaURL string
	aiService *AIService // 用来调用 Embedding API
}

func NewVectorService(chromaURL string, aiService *AIService) *VectorService {
	return &VectorService{
		chromaURL: chromaURL,
		aiService: aiService,
	}
}

// EnsureCollection 确保 Collection 存在（程序启动时调用一次）
func (s *VectorService) EnsureCollection() error {
	body := map[string]interface{}{
		"name": chromaCollection,
	}
	data, _ := json.Marshal(body)

	//我们在运行chroma服务以后，可以使用浏览器打开 http://localhost:8000/docs 即可看到chromaDB所有端点和交互界面。
	//使用 http.Post 向 ChromaDB 发送 POST 请求，路径是 /api/v1/collections。
	resp, err := http.Post(s.chromaURL+"/api/v1/collections", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("创建 Chroma Collection 失败: %w", err)
	}
	defer resp.Body.Close() //defer resp.Body.Close() 是 Go 的好习惯，防止内存泄漏。
	//resp 是 Go 的 http.Response 对象，代表 ChromaDB 对你 POST 请求的完整响应。
	/**http.Response 里面主要包含什么？
	type Response struct {
		StatusCode int                 // 状态码，例如 200、404、409
		Status     string              // 状态描述，例如 "200 OK"
		Proto      string              // "HTTP/1.1"
		Header     http.Header         // 响应头
		Body       io.ReadCloser       // 响应体（需要读取并 Close）
		ContentLength int64           // 内容长度
		// ... 还有 Request、TLS、Cookies 等字段
	}*/

	// 读取响应体
	b, _ := io.ReadAll(resp.Body)

	// 成功：200 / 201
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		fmt.Printf("✅ ChromaDB Collection [%s] 创建成功\n", chromaCollection)
		return nil
	}

	// 集合已存在：有些版本返回 409，有些返回 500 + UniqueConstraintError
	// 统一检查响应体内容来判断
	bodyStr := string(b)
	if strings.Contains(bodyStr, "UniqueConstraintError") ||
		strings.Contains(bodyStr, "already exists") ||
		resp.StatusCode == 409 {
		fmt.Printf("✅ ChromaDB Collection [%s] 已存在，继续使用\n", chromaCollection)
		return nil
	}

	// 真正的错误
	return fmt.Errorf("创建 Collection 失败，状态码: %d, 响应: %s", resp.StatusCode, bodyStr)
}

// getCollectionID 获取 Collection 的内部 UUID（Chroma 需要用 UUID 操作）
// ChromaDB 在创建 Collection 时会给它分配一个UUID（一串很长的随机字符串），后续很多操作（如 add、query）必须使用这个 UUID，而不是名称。所以这个辅助方法非常重要。
func (s *VectorService) getCollectionID() (string, error) {
	resp, err := http.Get(s.chromaURL + "/api/v1/collections/" + chromaCollection)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //这是一行链式调用，功能是：把 HTTP 响应体中的 JSON 数据解析成 Go 的 map。
	//json.NewDecoder(resp.Body)它接收一个 io.Reader（能读取数据的对象），返回一个 *json.Decoder。作用：创建一个 JSON 解码器，专门用来从响应体中读取 JSON 数据。
	//.Decode(&result) 它会从 resp.Body 中读取 JSON 数据，然后反序列化（解析） 到传入的变量中。
	id, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("无法获取 collection ID")
	}
	return id, nil
}

// AddDocument 把文档切块、向量化后存入 ChromaDB
func (s *VectorService) AddDocument(docID uint, content string) error { //参数：docID：文档的唯一 ID content：文档的原始文本（支持中文）
	fmt.Printf("📄 [文档入库] 开始处理文档 ID: %d，原始字符数: %d\n", docID, len([]rune(content))) //用 []rune 正确统计中文字符数

	// ========== 阶段一：文本切块 ==========
	chunks := splitIntoChunks(content, 500, 50) //这是 RAG（Retrieval-Augmented Generation）系统中非常关键的文本切块函数。
	//splitIntoChunks(text string, chunkSize, overlap int) []string
	//text：要切的原始长文本（支持中文）
	//chunkSize：每块目标字符数（示例中用 500）
	//overlap：相邻两块之间的重叠字符数（示例中用 50）//为什么要重叠：重要信息出现在块的边界时，仍能在两个块中都出现，显著提高检索质量（RAG 常用技巧
	//返回值：[]string —— 切好的一组文本块

	if len(chunks) == 0 {
		fmt.Printf("⚠️  [文本切块] 文档 %d 内容为空，跳过入库\n", docID)
		return nil
	}
	fmt.Printf("✅ [文本切块] 文档 %d 切块完成，共 %d 块（每块约500字，重叠50字）\n", docID, len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("   块 %02d: %d 字 | 预览: %.40s...\n", i+1, len([]rune(chunk)), chunk)
	}

	// ========== 阶段二：逐块向量化 ==========
	fmt.Printf("🔢 [向量化] 开始对 %d 个块调用 Embedding API...\n", len(chunks))
	embeddings := make([][]float32, 0, len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("   → 正在向量化块 %02d / %02d ...\n", i+1, len(chunks))
		vec, err := s.aiService.GetEmbedding(chunk) //循环对每个 chunk 调用 Embedding 接口
		if err != nil {
			fmt.Printf("❌ [向量化] 块 %02d 失败: %v\n", i+1, err)
			return fmt.Errorf("向量化失败 (块 %d): %w", i+1, err)
		}
		embeddings = append(embeddings, vec) //把每一块都添加到embedding中，循环结束，embedding就是全部向量化的向量
		fmt.Printf("   ✓ 块 %02d 向量化成功，维度: %d\n", i+1, len(vec))
	}
	fmt.Printf("✅ [向量化] 全部 %d 个块向量化完成\n", len(chunks))

	// ========== 阶段三：批量写入 ChromaDB ==========
	ids := make([]string, len(chunks))
	//metadatas 是 元数据数组（metadata = 数据的数据）。
	//它是一个 []map[string]interface{} 类型，也就是一组 map，长度和 chunks 完全一样。
	//每个块（chunk）对应一个 map，用于存储这个向量块的附加信息。
	metadatas := make([]map[string]interface{}, len(chunks))
	for i := range chunks {
		ids[i] = fmt.Sprintf("doc_%d_chunk_%d", docID, i)
		metadatas[i] = map[string]interface{}{
			"doc_id":      docID,
			"chunk_index": i,
		}
	}
	//获取 Collection UUID
	collectionID, err := s.getCollectionID()
	if err != nil {
		fmt.Printf("❌ [ChromaDB] 获取 Collection ID 失败: %v\n", err)
		return err
	}

	fmt.Printf("📦 [ChromaDB] 开始批量写入 %d 个向量块...\n", len(chunks))
	body := map[string]interface{}{
		"ids":        ids,        //向量唯一标识
		"embeddings": embeddings, //向量数据
		"documents":  chunks,     //原始文本
		"metadatas":  metadatas,  //附加结构化信息
	}
	data, _ := json.Marshal(body) //序列化数据，因为我们调用chromaDB的时候使用的是http请求，所以发送请求都要发送json序列的数据

	resp, err := http.Post(
		s.chromaURL+"/api/v1/collections/"+collectionID+"/add",
		"application/json",
		bytes.NewBuffer(data), //创建一个 bytes.Buffer 对象，把 data（JSON 字节）包装成一个 io.Reader，供 http.Post 使用。
	)
	if err != nil {
		fmt.Printf("❌ [ChromaDB] 写入请求失败: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body) //把 HTTP 响应的 Body（响应内容）全部读取出来 用于返回日志
		fmt.Printf("❌ [ChromaDB] 写入失败，状态码: %d，响应: %s\n", resp.StatusCode, string(b))
		return fmt.Errorf("向量存储失败: %s", string(b))
	}

	fmt.Printf("✅ [ChromaDB] 写入成功，文档 %d 共 %d 块已入库\n", docID, len(chunks))
	fmt.Printf("🎉 [文档入库] 文档 %d 全流程完成 ✓\n\n", docID)
	return nil
}

// Search 搜索最相关的 N 个文本块
func (s *VectorService) Search(query string, topN int) ([]string, error) {
	//query：用户的问题或搜索关键词（例如：“什么是向量数据库？”）
	//topN：要返回最相似的块数量（例如 5）
	//返回值：[]string（最相关的文本块列表） + error
	vec, err := s.aiService.GetEmbedding(query) //把用户查询转成向量（和文档块用同一个 Embedding 模型）
	if err != nil {
		return nil, fmt.Errorf("查询向量化失败: %w", err)
	}

	collectionID, err := s.getCollectionID()
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"query_embeddings": [][]float32{vec},      //必须是 [][]float32二维数组，即使只有一个 query，也要包一层 []
		"n_results":        topN,                  // 返回前 N 个最相似
		"include":          []string{"documents"}, // 只返回原始文本
	}
	data, _ := json.Marshal(body)

	resp, err := http.Post(
		s.chromaURL+"/api/v1/collections/"+collectionID+"/query",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// 解析返回的 documents[0]（第一个查询的结果）
	//ChromaDB 的 /query 接口返回的数据是支持批量查询的，所以结构是三层嵌套：
	//第1步：取出 documents 字段
	docs, ok := result["documents"].([]interface{}) //使用类型断言 .( []interface{} ) 把值转成 []interface{} 类型
	if !ok || len(docs) == 0 {
		return []string{}, nil
	} //检查是否合法
	//第2步：取出docs中第一个 query 的结果
	firstQuery, ok := docs[0].([]interface{})
	if !ok {
		return []string{}, nil
	}
	//第3步：把字符串提取出来
	chunks := make([]string, 0, len(firstQuery))
	for _, d := range firstQuery {
		if s, ok := d.(string); ok {
			chunks = append(chunks, s)
		}
	}
	return chunks, nil
}

// splitIntoChunks 把长文本按字符数切块，支持重叠（overlap）
// 把一篇很长的文本，按照固定长度切成多个小块（Chunk），并且相邻两块之间保留一部分重叠内容。这是 RAG（检索增强生成）系统中最基础、最重要的文本预处理步骤。
// 为什么需要 Overlap？
// 没有重叠：关键信息可能刚好被切在两块交界处，检索时容易丢失。
// 有重叠：上下文连续性更好，LLM 理解质量显著提升（50~100 字重叠是常见经验值）。
func splitIntoChunks(text string, chunkSize, overlap int) []string {
	runes := []rune(text) // 用 rune 支持中文 把字符串转为 []rune，每个中文字符、英文字符、标点都算1 个单位。
	var chunks []string
	step := chunkSize - overlap //假如说：step = 500 - 50 = 450  意思是：每向前移动 450 个字符，就开始切新的一块（从而产生 50 字重叠）
	if step <= 0 {
		step = chunkSize
	}
	for i := 0; i < len(runes); i += step {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes) // 最后一块可能不满 500 字
		}
		chunks = append(chunks, string(runes[i:end]))
		if end == len(runes) { // 已经切到结尾，结束循环
			break
		}
	}
	return chunks
}
