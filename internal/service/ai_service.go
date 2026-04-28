package service

/*
*openai.Client 具体是什么类型？我们到底创造了一个什么？
*openai.Client 的本质：
	Goclient *openai.Client     // ← AIService 持有的字段
	它是由 github.com/sashabaranov/go-openai 这个 SDK 提供的一个结构体指针。
我们实际创造的是什么？
	我们创造了一个可以和 DeepSeek（或其他 OpenAI 兼容接口）进行 HTTP 通信的客户端对象。
简单来说：
	它是一个封装好的 HTTP 客户端（Client）
内部包含了：
	API Key（身份认证）
	BaseURL（请求地址：https://api.deepseek.com）
	HTTP Transport（连接池、超时、重试等设置）
	各种方法：CreateChatCompletion、CreateEmbedding、CreateImage 等*/
import (
	"LeoAi/config"
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	client *openai.Client
	//定义结构体，只持有一个 *openai.Client（官方 OpenAI Go SDK）。
	//通过这个 client 可以调用 DeepSeek（因为 DeepSeek 兼容 OpenAI 接口）。
	embeddingClient *openai.Client // 硅基流动，用于向量化
}

func NewAIService(cfg *config.Config) *AIService { //传入的是config包中的config结构体类型的参数
	// 加这行，启动时看控制台输出
	fmt.Printf("🔍 [AIService] SiliconFlowKey=%q, SiliconFlowURL=%q\n", cfg.SiliconFlowKey, cfg.SiliconFlowURL)

	config := openai.DefaultConfig(cfg.DeepSeekKey)
	/*	作用：创建一个默认的 Client 配置对象（ClientConfig）。
		输入：DeepSeek 的 API Key（字符串）
		输出：一个 openai.ClientConfig 结构体，里面已经预设好了 OpenAI 官方默认参数。

		DefaultConfig 内部帮你做了这些事：
		把传入的 Key 设置到 config.APIKey
		设置默认的 BaseURL 为 OpenAI 官方地址（https://api.openai.com/v1）
		设置一些合理的默认 HTTP 配置（超时、重试等）*/
	config.BaseURL = "https://api.deepseek.com"
	/*	作用：把请求地址从 OpenAI 官方改成 DeepSeek 的官方地址。

		默认情况下 DefaultConfig 会指向 https://api.openai.com/v1
		改成 DeepSeek 的地址后，所有请求（ChatCompletion、Embedding 等）都会发到 DeepSeek 的服务器*/

	// 硅基流动 client（新增）
	sfCfg := openai.DefaultConfig(cfg.SiliconFlowKey)
	sfCfg.BaseURL = cfg.SiliconFlowURL
	return &AIService{
		client: openai.NewClientWithConfig(config),
		/*	作用：使用你自定义好的 config，真正创建一个可用的 OpenAI Client。
			输入：Config（包含 Key、BaseURL、HTTPClient 等）
			输出：*openai.Client 对象（后面用来调用 CreateChatCompletion 等方法）*/
		//openai.NewClient(apiKey)方法只能用官方 OpenAI
		//NewClientWithConfig(config)方法兼容 DeepSeek、Azure、Groq 等
		embeddingClient: openai.NewClientWithConfig(sfCfg),
	}
}

func (s *AIService) GenerateContent(topic string) (string, error) {
	// 1. 定义高质量 System Prompt（建议放在常量或单独变量里）
	systemPrompt := `你现在是顶级中文内容专家「Leo」，拥有15年媒体和自媒体写作经验。
你的写作原则：
1. 极致读者导向：始终站在读者角度思考他们真正想要什么。
2. 结构化输出：使用清晰的层级标题，用 emoji 作为每个大段的视觉标记，标题用加粗文字，配合 emoji，大量使用列表和表格。
3. 干货密度高：每一段都要有价值，不说废话。
4. 语言魅力：节奏感强，善用短句和排比，阅读体验优秀。
5. 真实性：基于事实和逻辑，不编造。
6. 输出格式：只返回完整的 Markdown 格式文章，不要任何前言后语。
7. 标题要求：主标题要吸引眼球且带 emoji，副标题用**粗体**说明核心价值。
8. 结尾要有行动号召或总结金句。`
	userPrompt := fmt.Sprintf(`主题：%s
请严格按照以下要求创作：
1. 字数控制在800-1000字之间
2. 使用清晰的 Markdown 结构（## 二级标题、### 三级标题）
3. 每部分都要包含实用价值或可执行建议
4. 语言生动但不夸张，适合普通读者阅读
5. 结尾增加1-2条具体行动建议

现在开始创作。`, topic)
	resp, err := s.client.CreateChatCompletion(
		context.Background(), //参数1：context.Background()提供一个上下文（Context），用于控制请求的生命周期。
		openai.ChatCompletionRequest{ //参数2：openai.ChatCompletionRequest{ ... }这是请求体（Request），你把要发给 AI 的所有指令都写在这里。
			Model: "deepseek-v4-flash",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			Temperature: 0.7,
		},
	)

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

// GetEmbedding 把文本转成向量（用于 RAG）
func (s *AIService) GetEmbedding(text string) ([]float32, error) {
	resp, err := s.embeddingClient.CreateEmbeddings( //s.embeddingClient：这是 OpenAI Go SDK 的客户端（*openai.Client）	//CreateEmbeddings：调用 Embedding 接口
		context.Background(), //默认上下文
		openai.EmbeddingRequest{ //请求结构体
			Input: []string{text},                       //传入的文本
			Model: openai.EmbeddingModel("BAAI/bge-m3"), // ← 模型名
		},
	)
	if err != nil {
		return nil, fmt.Errorf("获取 Embedding 失败: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("Embedding 返回为空")
	}
	return resp.Data[0].Embedding, nil
	//OpenAI Go SDK中的结构体：
	/*type EmbeddingResponse struct {
		Object string         `json:"object"`
		Data   []EmbeddingData `json:"data"`     // ← 重点在这里
		Model  string         `json:"model"`
		Usage  Usage          `json:"usage"`
	}*/
	//可以看到其中的data字段是一个[]EmbeddingData类型的切片 所以我们哪怕只传一条数据，也要用data[0]
}

// RAGGenerate 基于检索到的上下文生成回答
func (s *AIService) RAGGenerate(question string, contextChunks []string) (string, error) {
	/*	question：用户的问题
		contextChunks：从 ChromaDB 检索回来的最相关文本块（Search 方法返回的）
		返回：大模型生成的回答 + 错误信息*/

	// 拼接上下文
	contextText := ""
	for i, chunk := range contextChunks {
		contextText += fmt.Sprintf("\n【参考片段 %d】\n%s\n", i+1, chunk)
	}

	systemPrompt := `你是一个专业的文档问答助手。
请严格基于用户提供的【参考资料】来回答问题。
规则：
1. 只使用参考资料中的信息作答，不要编造。
2. 如果参考资料中没有相关信息，如实告知"文档中未找到相关信息"。
3. 回答要简洁、准确，并在回答后注明参考了哪个片段。
4. 使用 Markdown 格式，让回答清晰易读。`

	userPrompt := fmt.Sprintf("参考资料：%s\n\n用户问题：%s", contextText, question)

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "deepseek-v4-flash",
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: userPrompt},
			},
			Temperature: 0.3, // RAG 场景用低温度，减少幻觉
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
