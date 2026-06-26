package unified

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Coordinator المنسق المركزي بين جميع الأنظمة
type Coordinator struct {
	unifiedAgent *UnifiedAgent
	logger       *zap.Logger
	mu           sync.RWMutex
}

// NewCoordinator ينشئ منسق مركزي جديد
func NewCoordinator(logger *zap.Logger) *Coordinator {
	return &Coordinator{
		logger: logger,
	}
}

// Initialize يهيئ المنسق
func (c *Coordinator) Initialize(ctx context.Context, unifiedAgent *UnifiedAgent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.unifiedAgent = unifiedAgent
	c.logger.Info("تم تهيئة المنسق المركزي")
	return nil
}

// ExecuteTask ينفذ مهمة باستخدام جميع الأنظمة المنسقة
func (c *Coordinator) ExecuteTask(ctx context.Context, executionContext *ExecutionContext) (*UnifiedTaskResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// [WHY] تنسيق تنفيذ المهمة بين جميع الأنظمة
	// [HOW] يستخدم جميع الأنظمة بشكل متناسق
	// [SAFETY] يضمن عدم وجود تعارضات

	result := &UnifiedTaskResult{
		Task:     executionContext.Task,
		Success:  false,
		Metadata: make(map[string]interface{}),
	}

	// استخدام النظام الجماعي للتنفيذ
	collectiveResult, err := c.unifiedAgent.collectiveSystem.ExecuteTask(ctx, executionContext.Task, c.unifiedAgent.agentID)
	if err != nil {
		return nil, fmt.Errorf("فشل تنفيذ المهمة في النظام الجماعي: %w", err)
	}

	success, ok := collectiveResult["success"].(bool)
	if !ok {
		success = false
	}
	result.Success = success
	result.Output = collectiveResult
	confidence, ok := collectiveResult["confidence"].(float64)
	if !ok {
		confidence = 0.0
	}
	result.Confidence = confidence
	result.Metadata["collective_result"] = collectiveResult

	return result, nil
}

// GetSummary يحصل على ملخص المنسق
func (c *Coordinator) GetSummary() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"initialized": c.unifiedAgent != nil,
		"active":      true,
	}
}

// ExecutionContext سياق التنفيذ
type ExecutionContext struct {
	Task       string
	Context    map[string]interface{}
	Execution  *Execution
}

// Execution يمثل تنفيذ
type Execution struct {
	ID       string
	Task     string
	Progress float64
	State    map[string]interface{}
}
