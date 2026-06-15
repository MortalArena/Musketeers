package adapters

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// CLIAdapter محول لـ CLI (سطر الأوامر)
type CLIAdapter struct {
	info       *agent.AgentInfo
	command    string
	args       []string
	logger     *zap.Logger
	available  bool
}

// CLIConfig إعدادات CLI
type CLIConfig struct {
	Command string
	Args    []string
	Name    string
}

// NewCLIAdapter ينشئ محول CLI جديد
func NewCLIAdapter(config *CLIConfig) *CLIAdapter {
	return &CLIAdapter{
		info: &agent.AgentInfo{
			ID:            fmt.Sprintf("cli_%s", config.Name),
			Name:          fmt.Sprintf("%s CLI Agent", config.Name),
			Type:          agent.AgentTypeCLI,
			Provider:      "local",
			Model:         config.Command,
			Version:       "1.0.0",
			Endpoint:      "",
			AuthMethod:    "none",
			MaxTokens:     4096,
			ContextWindow: 8192,
			CreatedAt:     time.Now(),
		},
		command:   config.Command,
		args:      config.Args,
		logger:    zap.NewNop(),
		available: true,
	}
}

// SetLogger يضبط logger
func (ca *CLIAdapter) SetLogger(logger *zap.Logger) {
	ca.logger = logger
}

// GetInfo يعيد معلومات الوكيل
func (ca *CLIAdapter) GetInfo() *agent.AgentInfo {
	return ca.info
}

// SendMessage يرسل رسالة للوكيل
func (ca *CLIAdapter) SendMessage(ctx context.Context, prompt string) (*agent.AgentResponse, error) {
	startTime := time.Now()

	// تجهيز الأوامر
	args := append(ca.args, prompt)

	// تنفيذ الأمر
	cmd := exec.CommandContext(ctx, ca.command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("CLI command failed: %w, output: %s", err, string(output))
	}

	duration := time.Since(startTime)

	ca.logger.Info("CLI message sent",
		zap.String("command", ca.command),
		zap.Int("output_length", len(output)),
		zap.Duration("duration", duration),
	)

	return &agent.AgentResponse{
		Content:  strings.TrimSpace(string(output)),
		Tokens:   len(strings.Split(string(output), " ")),
		Duration: duration,
	}, nil
}

// ExecuteTask ينفذ مهمة
func (ca *CLIAdapter) ExecuteTask(ctx context.Context, task *agent.AgentTask) (*agent.TaskExecutionResult, error) {
	startTime := time.Now()

	// تجهيز prompt من المهمة
	prompt := fmt.Sprintf("%s: %s", task.Title, task.Description)
	if task.Context != "" {
		prompt += fmt.Sprintf(" (%s)", task.Context)
	}

	// إرسال الرسالة
	response, err := ca.SendMessage(ctx, prompt)
	if err != nil {
		return &agent.TaskExecutionResult{
			Success: false,
			Error:   err.Error(),
			Duration: time.Since(startTime),
		}, nil
	}

	duration := time.Since(startTime)

	ca.logger.Info("CLI task executed",
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
func (ca *CLIAdapter) GetCapabilities() []agent.AgentCapability {
	return []agent.AgentCapability{
		agent.CapabilityCodeGeneration,
		agent.CapabilityCodeReview,
		agent.CapabilityTesting,
	}
}

// GetStatus يعيد حالة الوكيل
func (ca *CLIAdapter) GetStatus() *agent.AgentStatus {
	return &agent.AgentStatus{
		IsAvailable:  ca.available,
		CurrentTask:  "",
		Load:         0,
		LastSeen:     time.Now(),
		ResponseTime: 200 * time.Millisecond,
		SuccessRate:  1.0,
		TotalTasks:   0,
		FailedTasks:  0,
	}
}

// IsAvailable يعيد ما إذا كان الوكيل متاحاً
func (ca *CLIAdapter) IsAvailable() bool {
	return ca.available
}

// Close يغلق الوكيل
func (ca *CLIAdapter) Close() error {
	ca.available = false
	ca.logger.Info("CLI adapter closed",
		zap.String("command", ca.command),
	)
	return nil
}
