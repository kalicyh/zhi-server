package zhipu

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "time"
)

const (
    DefaultAPIURL = "https://open.bigmodel.cn/api/paas/v4/images/generations"
    DefaultModel  = "cogview-3-flash" // 默认使用cogview-3-flash模型
)

// ZhipuProvider 智普AI文生图提供者
type ZhipuProvider struct {
    APIKey  string
    APIURL  string
    Model   string
    Timeout time.Duration
}

// NewZhipuProvider 创建一个新的智普提供者
func NewZhipuProvider(apiKey string, options ...func(*ZhipuProvider)) *ZhipuProvider {
    provider := &ZhipuProvider{
        APIKey:  apiKey,
        APIURL:  DefaultAPIURL,
        Model:   DefaultModel,
        Timeout: 60 * time.Second,
    }

    // 应用可选配置
    for _, option := range options {
        option(provider)
    }

    return provider
}

// WithModel 设置模型
func WithModel(model string) func(*ZhipuProvider) {
    return func(p *ZhipuProvider) {
        p.Model = model
    }
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) func(*ZhipuProvider) {
    return func(p *ZhipuProvider) {
        p.Timeout = timeout
    }
}

// ImageRequest 文生图请求结构
type ImageRequest struct {
    Model    string `json:"model"`             // 必填，模型编码
    Prompt   string `json:"prompt"`            // 必填，图像描述
    Quality  string `json:"quality,omitempty"` // 可选，图像质量 (hd/standard)
    Size     string `json:"size,omitempty"`    // 可选，图片尺寸
    UserID   string `json:"user_id,omitempty"` // 可选，终端用户ID
}

// ImageResponseData 图像响应数据
type ImageResponseData struct {
    URL string `json:"url"` // 生成的图片URL
}

// ContentFilterInfo 内容过滤信息
type ContentFilterInfo struct {
    Role  string `json:"role"`  // 安全生效环节
    Level int    `json:"level"` // 严重程度
}

// ImageResponse 文生图响应结构
type ImageResponse struct {
    Created       int64              `json:"created"`        // 创建时间戳（Unix时间戳）
    Data          []ImageResponseData `json:"data"`           // 图片数据
    ContentFilter []ContentFilterInfo `json:"content_filter"` // 内容过滤信息
    Error         *struct {
        Message string `json:"message"`
        Code    string `json:"code"`
    } `json:"error,omitempty"`
}

// GenerateImage 生成图片
// prompt: 图像描述
// quality: 生成图像的质量，默认为 standard
//			hd : 生成更精细、细节更丰富的图像，整体一致性更高，耗时约20 秒
//			standard :快速生成图像，适合对生成速度有较高要求的场景，耗时约 5-10 秒
//			此参数仅支持cogview-4-250304 。
// Size:图片尺寸，推荐枚举值：1024x1024,768x1344,864x1152,1344x768,1152x864,1440x720,720x1440，默认是1024x1024。
// 		自定义参数：长宽均需满足 512px - 2048px 之间，需被16整除, 并保证最大像素数不超过 2^21 px。
// UserID 终端用户的唯一ID，协助平台对终端用户的违规行为、生成违法及不良信息或其他滥用行为进行干预。ID长度要求：最少6个字符，最多128个字符。
func (p *ZhipuProvider) GenerateImage(prompt string, quality string, size string, userID string) (*ImageResponse, error) {
    request := ImageRequest{
        Model:   p.Model,
        Prompt:  prompt,
        Quality: quality,
        Size:    size,
        UserID:  userID,
    }

    jsonData, err := json.Marshal(request)
    if err != nil {
        return nil, fmt.Errorf("json 编码错误: %w", err)
    }

    // 创建HTTP请求
    req, err := http.NewRequest("POST", p.APIURL, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("创建请求失败: %w", err)
    }

    // 设置请求头
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))

    // 创建HTTP客户端并发送请求
    client := &http.Client{
        Timeout: p.Timeout,
    }
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("API请求失败: %w", err)
    }
    defer resp.Body.Close()

    // 读取响应数据
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取响应失败: %w", err)
    }

    // 解析JSON响应
    var response ImageResponse
	fmt.Println(string(body))
    err = json.Unmarshal(body, &response)
    if err != nil {
        return nil, fmt.Errorf("JSON解析错误: %w", err)
    }

    // 检查错误
    if response.Error != nil {
        return nil, errors.New(response.Error.Message)
    }

    // 检查是否有图片数据
    if len(response.Data) == 0 {
        return nil, errors.New("未返回图片数据")
    }

    return &response, nil
}