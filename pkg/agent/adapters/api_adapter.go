package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// APIAdapter محول لـ REST API (Claude, OpenAI, Gemini)
type APIAdapter struct {
	info     *agent.AgentInfo
	client   *http.Client
	apiKey   string
	baseURL  string
	model    string
	logger   *zap.Logger
	available bool
}

// APIConfig إعدادات API
type APIConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	MaxTokens   int
	Timeout     time.Duration
}

// NewAPIAdapter ينشئ محول API جديد
func NewAPIAdapter(config *APIConfig) *APIAdapter {
	return &APIAdapter{
		info: &agent.AgentInfo{
			ID:            fmt.Sprintf("api_%s", config.Model),
			Name:          fmt.Sprintf("%s API Agent", config.Model),
			Type:          agent.AgentTypeAPI,
			Provider:      detectProvider(config.BaseURL),
			Model:         config.Model,
			Version:       "1.0.0",
			Endpoint:      config.BaseURL,
			AuthMethod:    "api_key",
			MaxTokens:     config.MaxTokens,
			ContextWindow: 200000,
			CreatedAt:     time.Now(),
		},
		client: &http.Client{
			Timeout: config.Timeout,
		},
		apiKey:   config.APIKey,
		baseURL:  config.BaseURL,
		model:    config.Model,
		logger:   zap.NewNop(),
		available: true,
	}
}

// detectProvider يكتشف المزود من الرابط
func detectProvider(baseURL string) string {
	if baseURL == "" {
		return "unknown"
	}
	
	switch {
	case contains(baseURL, "anthropic.com"):
		return "claude"
	case contains(baseURL, "openai.com"):
		return "openai"
	case contains(baseURL, "googleapis.com"):
		return "google"
	default:
		return "custom"
	}
}

// contains يتحقق من وجود نص في سلسلة
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SetLogger يضبط logger
func (aa *APIAdapter) SetLogger(logger *zap.Logger) {
	aa.logger = logger
}

// GetInfo يعيد معلومات الوكيل
func (aa *APIAdapter) GetInfo() *agent.AgentInfo {
	return aa.info
}

// SendMessage يرسل رسالة للوكيل
func (aa *APIAdapter) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	startTime := time.Now()

	// تجهيز الطلب
	requestBody := map[string]interface{}{
		"model": aa.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": aa.info.MaxTokens,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// إنشاء طلب HTTP
	req, err := http.NewRequestWithContext(ctx, "POST", aa.baseURL+"/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", aa.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// إرسال الطلب
	resp, err := aa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// قراءة الاستجابة
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// تحليل الاستجابة
	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	content := ""
	if len(response.Content) > 0 {
		content = response.Content[0].Text
	}

	duration := time.Since(startTime)

	aa.logger.Info("API message sent",
		zap.String("model", aa.model),
		zap.Int("input_tokens", response.Usage.InputTokens),
		zap.Int("output_tokens", response.Usage.OutputTokens),
		zap.Duration("duration", duration),
	)

	return &agent.AgentResponse{
		Content:  content,
		Tokens:   response.Usage.InputTokens + response.Usage.OutputTokens,
		Duration: duration,
		Metadata: map[string]interface{}{
			"input_tokens":  response.Usage.InputTokens,
			"output_tokens": response.Usage.OutputTokens,
		},
	}, nil
}

// ExecuteTask ينفذ مهمة
func (aa *APIAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	startTime := time.Now()

	// تجهيز prompt من المهمة
	prompt := fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description)
	if task.Context != "" {
		prompt += fmt.Sprintf("\nContext: %s", task.Context)
	}

	// إرسال الرسالة
	response, err := aa.SendMessage(ctx, prompt)
	if err != nil {
		return &agent.TaskExecutionResult{
			Success: false,
			Error:   err.Error(),
			Duration: time.Since(startTime),
		}, nil
	}

	duration := time.Since(startTime)

	aa.logger.Info("API task executed",
		zap.String("task_id", task.ID),
		zap.String("task_title", task.Title),
		zap.Bool("success", true),
		zap.Duration("duration", duration),
	)

	return &agent.TaskExecutionResult{
		Success:  true,
		Output:   response.Content,
		Duration: duration,
		Metrics: map[string]interface{}{
			"tokens": response.Tokens,
		},
	}, nil
}

// GetCapabilities يعيد قدرات الوكيل
func (aa *APIAdapter) GetCapabilities() []agent.AgentCapability {
	return []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
		agent.CapabilityDocumentation,
		agent.CapabilityAnalysis,
	}
}

// GetStatus يعيد حالة الوكيل
func (aa *APIAdapter) GetStatus() *agent.AgentStatus {
	return &agent.AgentStatus{
		IsAvailable:  aa.available,
		CurrentTask:  "",
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 100 * time.Millisecond,
		SuccessRate:  1.0,
		TotalTasks:   0,
		FailedTasks:  0,
	}
}

// IsAvailable يعيد ما إذا كان الوكيل متاحاً
func (aa *APIAdapter) IsAvailable() bool {
	return aa.available
}

// Close يغلق الوكيل
func (aa *APIAdapter) Close() error {
	aa.available = false
	aa.logger.Info("API adapter closed",
		zap.String("model", aa.model),
	)
	return nil
}
