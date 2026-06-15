package adapters

import (
	"context"
	"fmt"
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
}

// LocalConfig إعدادات النموذج المحلي
type LocalConfig struct {
	BaseURL string
	Model   string
	Name    string
}

// NewLocalAdapter ينشئ محول محلي جديد
func NewLocalAdapter(config *LocalConfig) *LocalAdapter {
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
			MaxTokens:     4096,
			ContextWindow: 8192,
			CreatedAt:     time.Now(),
		},
		baseURL:   config.BaseURL,
		model:     config.Model,
		logger:    zap.NewNop(),
		available: true,
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

	// محاكاة استجابة من النموذج المحلي
	// في التطبيق الحقيقي، سيتم الاتصال بـ Ollama API
	response := fmt.Sprintf("Local model %s response to: %s", la.model, prompt)

	duration := time.Since(startTime)

	la.logger.Info("Local message sent",
		zap.String("model", la.model),
		zap.Int("prompt_length", len(prompt)),
		zap.Duration("duration", duration),
	)

	return &agent.AgentResponse{
		Content:  response,
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
			Success: false,
			Error:   err.Error(),
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
