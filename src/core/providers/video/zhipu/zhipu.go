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
	// API endpoints
	DefaultGenerateAPIURL = "https://open.bigmodel.cn/api/paas/v4/videos/generations"
	DefaultResultAPIURL   = "https://open.bigmodel.cn/api/paas/v4/async-result/%s"
	DefaultModel          = "cogvideox-flash" // 默认使用cogvideox-flash模型

	// Supported resolutions
	Resolution720P  = "720x480"
	Resolution1K    = "1024x1024"
	Resolution1280P = "1280x960"
	Resolution960P  = "960x1280"
	ResolutionHD    = "1920x1080"
	ResolutionHDV   = "1080x1920"
	Resolution2K    = "2048x1080"
	Resolution4K    = "3840x2160"
)

// Supported resolutions list
var ValidResolutions = []string{
	Resolution720P, Resolution1K, Resolution1280P,
	Resolution960P, ResolutionHD, ResolutionHDV,
	Resolution2K, Resolution4K,
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ZhipuVideoProvider 智普AI视频生成提供者
type ZhipuVideoProvider struct {
	APIKey  string
	APIURL  string
	Model   string
	Timeout time.Duration
}

// NewZhipuVideoProvider 创建一个新的智普提供者
func NewZhipuVideoProvider(apiKey string, options ...func(*ZhipuVideoProvider)) *ZhipuVideoProvider {
	provider := &ZhipuVideoProvider{
		APIKey:  apiKey,
		APIURL:  DefaultGenerateAPIURL,
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
func WithModel(model string) func(*ZhipuVideoProvider) {
	return func(p *ZhipuVideoProvider) {
		p.Model = model
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) func(*ZhipuVideoProvider) {
	return func(p *ZhipuVideoProvider) {
		p.Timeout = timeout
	}
}

// VideoRequest 视频生成请求结构
type VideoRequest struct {
	Model     string `json:"model"`               // 必填，模型编码
	Prompt    string `json:"prompt,omitempty"`    // 可选，视频文本描述
	Quality   string `json:"quality,omitempty"`   // 可选，输出模式 (quality/speed)，注：cogvideox-flash不支持此参数
	WithAudio bool   `json:"with_audio"`          // 是否生成AI音效，支持场景元素识别并生成适配的音效组合
	ImageURL  string `json:"image_url,omitempty"` // 可选，基础图片URL
	Size      string `json:"size,omitempty"`      // 可选，支持分辨率: 720x480, 1024x1024, 1280x960, 960x1280, 1920x1080, 1080x1920, 2048x1080, 3840x2160
	FPS       int    `json:"fps,omitempty"`       // 可选，帧率 (支持 30/60)
	UserID    string `json:"user_id,omitempty"`   // 可选，终端用户ID
}

// VideoResponse 视频生成响应结构
type VideoResponse struct {
	RequestID  string `json:"request_id"`  // 任务请求ID
	ID         string `json:"id"`          // 任务ID
	Model      string `json:"model"`       // 使用的模型
	TaskStatus string `json:"task_status"` // 任务状态
}

// VideoResultItem 视频结果项
type VideoResultItem struct {
	URL           string `json:"url"`             // 视频URL
	CoverImageURL string `json:"cover_image_url"` // 视频封面URL
}

// VideoResult 视频生成结果
type VideoResult struct {
	Model       string            `json:"model"`        // 模型名称
	RequestID   string            `json:"request_id"`   // 请求ID
	TaskStatus  string            `json:"task_status"`  // 任务状态
	VideoResult []VideoResultItem `json:"video_result"` // 视频结果
}

// GenerateVideo 生成视频
func (p *ZhipuVideoProvider) GenerateVideo(request *VideoRequest) (*VideoResponse, error) {
	if request.Model == "" {
		request.Model = p.Model
	}

	if request.Prompt == "" && request.ImageURL == "" {
		return nil, errors.New("prompt or image_url is required")
	}

	// 验证参数
	if request.Model == "cogvideox-flash" && request.Quality != "" {
		return nil, errors.New("cogvideox-flash 模型不支持 quality 参数设置")
	}

	// 验证分辨率
	if request.Size != "" {
		isValid := false
		for _, res := range ValidResolutions {
			if request.Size == res {
				isValid = true
				break
			}
		}
		if !isValid {
			return nil, fmt.Errorf("无效的分辨率: %s, 支持的分辨率: %v", request.Size, ValidResolutions)
		}
	}

	// 验证帧率
	if request.FPS != 0 && request.FPS != 30 && request.FPS != 60 {
		return nil, fmt.Errorf("无效的帧率: %d, 仅支持 30 或 60", request.FPS)
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

	// 尝试解析错误响应
	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Code != "" {
		return nil, fmt.Errorf("API错误 [%s]: %s", errorResp.Error.Code, errorResp.Error.Message)
	}

	// 解析正常响应
	var response VideoResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("JSON解析错误: %w", err)
	}

	// 验证响应字段
	if response.ID == "" {
		return nil, errors.New("无效的响应：缺少任务ID")
	}

	return &response, nil
}

// GetVideoResult 获取视频生成结果
func (p *ZhipuVideoProvider) GetVideoResult(taskID string) (*VideoResult, error) {
	if taskID == "" {
		return nil, errors.New("无效的任务ID")
	}

	// 构建结果查询URL
	url := fmt.Sprintf(DefaultResultAPIURL, taskID)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
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

	// 尝试解析错误响应
	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Code != "" {
		return nil, fmt.Errorf("API错误 [%s]: %s", errorResp.Error.Code, errorResp.Error.Message)
	}

	// 解析正常响应
	var result VideoResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("JSON解析错误: %w", err)
	}

	// 验证响应
	if result.TaskStatus == "" {
		return nil, errors.New("无效的响应：缺少任务状态")
	}

	return &result, nil
}
