package types

import "context"

// Message 对话消息结构
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall 工具调用结构
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// Function 函数定义结构
type Function struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  FunctionParams `json:"parameters"`
	Required    []string       `json:"required,omitempty"`
}

// FunctionParams 函数参数定义
type FunctionParams struct {
	Type       string                 `json:"type"`
	Properties map[string]ParamSchema `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// ParamSchema 参数模式定义
type ParamSchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// FunctionCall 函数调用结果
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	ID        string `json:"id,omitempty"`
}

// Response LLM响应结构
type Response struct {
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	StopReason string     `json:"stop_reason,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// Provider 基础提供者接口
type Provider interface {
	Initialize() error
	Cleanup() error
}

// LLMProvider 大语言模型提供者接口
type LLMProvider interface {
	Provider
	Response(ctx context.Context, sessionID string, messages []Message) (<-chan string, error)
	ResponseWithFunctions(ctx context.Context, sessionID string, messages []Message, functions []Function) (<-chan Response, error)
}
