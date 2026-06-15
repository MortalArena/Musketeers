package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// IDEAdapter محول لـ IDE (VS Code, JetBrains)
type IDEAdapter struct {
	info      *agent.AgentInfo
	ideType   string
	logger    *zap.Logger
	available bool
}

// IDEConfig إعدادات IDE
type IDEConfig struct {
	IDEType string // vscode, jetbrains
	Name    string
}

// NewIDEAdapter ينشئ محول IDE جديد
func NewIDEAdapter(config *IDEConfig) *IDEAdapter {
	return &IDEAdapter{
		info: &agent.AgentInfo{
			ID:            fmt.Sprintf("ide_%s", config.IDEType),
			Name:          fmt.Sprintf("%s IDE Agent", config.Name),
			Type:          agent.AgentTypeIDE,
			Provider:      config.IDEType,
			Model:         "ide-integration",
			Version:       "1.0.0",
			Endpoint:      "",
			AuthMethod:    "none",
			MaxTokens:     4096,
			ContextWindow: 8192,
			CreatedAt:     time.Now(),
		},
		ideType:   config.IDEType,
		logger:    zap.NewNop(),
		available: true,
	}
}

// SetLogger يضبط logger
func (ia *IDEAdapter) SetLogger(logger *zap.Logger) {
	ia.logger = logger
}

// GetInfo يعيد معلومات الوكيل
func (ia *IDEAdapter) GetInfo() *agent.AgentInfo {
	return ia.info
}

// SendMessage يرسل رسالة للوكيل
func (ia *IDEAdapter) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	startTime := time.Now()

	// محاكاة استجابة من IDE
	response := fmt.Sprintf("IDE %s response: %s", ia.ideType, prompt)

	duration := time.Since(startTime)

	ia.logger.Info("IDE message sent",
		zap.String("ide_type", ia.ideType),
		zap.Int("prompt_length", len(prompt)),
		zap.Duration("duration", duration),
	)

	return &agent.AgentResponse{
		Content:  response,
		Tokens:   len(prompt) / 4,
		Duration: duration,
	}, nil
}

// ExecuteTask ينفذ مهمة
func (ia *IDEAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	startTime := time.Now()

	// تجهيز prompt من المهمة
	prompt := fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description)
	if task.Context != "" {
		prompt += fmt.Sprintf("\nContext: %s", task.Context)
	}

	// إرسال الرسالة
	response, err := ia.SendMessage(ctx, prompt)
	if err != nil {
		return &agent.TaskExecutionResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, nil
	}

	duration := time.Since(startTime)

	ia.logger.Info("IDE task executed",
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
func (ia *IDEAdapter) GetCapabilities() []agent.AgentCapability {
	return []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
		agent.CapabilityFileOperations,
		agent.CapabilityTerminalAccess,
	}
}

// GetStatus يعيد حالة الوكيل
func (ia *IDEAdapter) GetStatus() *agent.AgentStatus {
	return &agent.AgentStatus{
		IsAvailable:  ia.available,
		CurrentTask:  "",
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 150 * time.Millisecond,
		SuccessRate:  1.0,
		TotalTasks:   0,
		FailedTasks:  0,
	}
}

// IsAvailable يعيد ما إذا كان الوكيل متاحاً
func (ia *IDEAdapter) IsAvailable() bool {
	return ia.available
}

// Close يغلق الوكيل
func (ia *IDEAdapter) Close() error {
	ia.available = false
	ia.logger.Info("IDE adapter closed",
		zap.String("ide_type", ia.ideType),
	)
	return nil
}
