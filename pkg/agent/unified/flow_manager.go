package unified

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// FlowManager مدير تدفق البيانات بين الأنظمة
type FlowManager struct {
	unifiedAgent *UnifiedAgent
	logger       *zap.Logger
	mu           sync.RWMutex
}

// NewFlowManager ينشئ مدير تدفق جديد
func NewFlowManager(logger *zap.Logger) *FlowManager {
	return &FlowManager{
		logger: logger,
	}
}

// Initialize يهيئ مدير التدفق
func (fm *FlowManager) Initialize(ctx context.Context, unifiedAgent *UnifiedAgent) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.unifiedAgent = unifiedAgent
	fm.logger.Info("تم تهيئة مدير التدفق")
	return nil
}

// CreateExecutionContext ينشئ سياق تنفيذ
func (fm *FlowManager) CreateExecutionContext(ctx context.Context, task string) *ExecutionContext {
	return &ExecutionContext{
		Task:    task,
		Context: make(map[string]interface{}),
		Execution: &Execution{
			ID:       generateID(),
			Task:     task,
			Progress: 0.0,
			State:    make(map[string]interface{}),
		},
	}
}

// GetSummary يحصل على ملخص مدير التدفق
func (fm *FlowManager) GetSummary() map[string]interface{} {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	return map[string]interface{}{
		"initialized": fm.unifiedAgent != nil,
		"active":      true,
	}
}

// generateID ينشئ معرف فريد
func generateID() string {
	return "exec_" + randomString(8)
}

// randomString ينشئ سلسلة عشوائية
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
