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

// LocalAdapter محول للنماذج المحلية (Ollama, LocalAI)
type LocalAdapter struct {
	info      *agent.AgentInfo
	baseURL   string
	model     string
	logger    *zap.Logger
	available bool
	client    *http.Client
	timeout   time.Duration
	maxTokens int
}

// LocalConfig إعدادات النموذج المحلي
type LocalConfig struct {
	BaseURL   string
	Model     string
	Name      string
	Timeout   time.Duration
	MaxTokens int
}

// OllamaRequest هيكل طلب Ollama
type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// OllamaResponse هيكل استجابة Ollama
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// NewLocalAdapter ينشئ محول محلي جديد
func NewLocalAdapter(config *LocalConfig) *LocalAdapter {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434" // [WHY] الافتراضي لـ Ollama
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute // [WHY] مهلة افتراضية
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096 // [WHY] الحد الأقصى الافتراضي
	}

	return &LocalAdapter{
		info: &agent.AgentInfo{
			ID:            fmt.Sprintf("local_%s", config.Model),
			Name:          fmt.Sprintf("%s Local Agent", config.Name),
			Type:          agent.AgentTypeLocal,
			Provider:      "ollama",
			Model:         config.Model,
			Version:       "1.0.0",
			Endpoint:      config.BaseURL,
			AuthMethod:    "none",
			MaxTokens:     config.MaxTokens,
			ContextWindow: 8192,
			CreatedAt:     time.Now(),
		},
		baseURL:   config.BaseURL,
		model:     config.Model,
		logger:    zap.NewNop(),
		available: true,
		client:    &http.Client{Timeout: config.Timeout},
		timeout:   config.Timeout,
		maxTokens: config.MaxTokens,
	}
}

// SetLogger يضبط logger
func (la *LocalAdapter) SetLogger(logger *zap.Logger) {
	la.logger = logger
}

// GetInfo يعيد معلومات الوكيل
func (la *LocalAdapter) GetInfo() *agent.AgentInfo {
	return la.info
}

// SendMessage يرسل رسالة للوكيل
func (la *LocalAdapter) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	startTime := time.Now()

	// إنشاء طلب Ollama
	request := OllamaRequest{
		Model:  la.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"num_predict": la.maxTokens,
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("فشل ترميز الطلب: %w", err)
	}

	// إرسال الطلب إلى Ollama API
	url := fmt.Sprintf("%s/api/generate", la.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("فشل إنشاء الطلب: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := la.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("فشل الاتصال بـ Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("فشل الطلب من Ollama: %s - %s", resp.Status, string(body))
	}

	// قراءة الاستجابة
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("فشل فك ترميز الاستجابة: %w", err)
	}

	duration := time.Since(startTime)

	la.logger.Info("Local message sent",
		zap.String("model", la.model),
		zap.Int("prompt_length", len(prompt)),
		zap.Duration("duration", duration),
		zap.Int("response_length", len(ollamaResp.Response)),
	)

	return &agent.AgentResponse{
		Content:  ollamaResp.Response,
		Tokens:   len(prompt) / 4, // تقدير تقريبي
		Duration: duration,
	}, nil
}

// ExecuteTask ينفذ مهمة
func (la *LocalAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	startTime := time.Now()

	// تجهيز prompt من المهمة
	prompt := fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description)
	if task.Context != "" {
		prompt += fmt.Sprintf("\nContext: %s", task.Context)
	}

	// إرسال الرسالة
	response, err := la.SendMessage(ctx, prompt)
	if err != nil {
		return &agent.TaskExecutionResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, nil
	}

	duration := time.Since(startTime)

	la.logger.Info("Local task executed",
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
func (la *LocalAdapter) GetCapabilities() []agent.AgentCapability {
	return []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
		agent.CapabilityDocumentation,
		agent.CapabilityAnalysis,
	}
}

// GetStatus يعيد حالة الوكيل
func (la *LocalAdapter) GetStatus() *agent.AgentStatus {
	return &agent.AgentStatus{
		IsAvailable:  la.available,
		CurrentTask:  "",
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 500 * time.Millisecond,
		SuccessRate:  1.0,
		TotalTasks:   0,
		FailedTasks:  0,
	}
}

// IsAvailable يعيد ما إذا كان الوكيل متاحاً
func (la *LocalAdapter) IsAvailable() bool {
	return la.available
}

// Close يغلق الوكيل
func (la *LocalAdapter) Close() error {
	la.available = false
	la.logger.Info("Local adapter closed",
		zap.String("model", la.model),
	)
	return nil
}
