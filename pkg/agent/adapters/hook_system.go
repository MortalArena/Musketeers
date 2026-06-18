package adapters

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AgentHookSystem نظام الخطافات للوكيل
type AgentHookSystem struct {
	hooks   map[string][]Hook
	mu      sync.RWMutex
	logger  *zap.Logger
	enabled bool
}

// Hook واجهة الخطاف
type Hook interface {
	Execute(ctx context.Context, event HookEvent) error
	Name() string
	Priority() int
}

// HookEvent حدث الخطاف
type HookEvent struct {
	Type      string
	Timestamp time.Time
	Data      map[string]interface{}
	Source    string
}

// HookFunc دالة خطاف بسيطة
type HookFunc struct {
	name     string
	priority int
	fn       func(ctx context.Context, event HookEvent) error
}

// NewHookFunc ينشئ دالة خطاف جديدة
func NewHookFunc(name string, priority int, fn func(ctx context.Context, event HookEvent) error) *HookFunc {
	return &HookFunc{
		name:     name,
		priority: priority,
		fn:       fn,
	}
}

// Execute ينفذ الخطاف
func (hf *HookFunc) Execute(ctx context.Context, event HookEvent) error {
	return hf.fn(ctx, event)
}

// Name يرجع اسم الخطاف
func (hf *HookFunc) Name() string {
	return hf.name
}

// Priority يرجع أولوية الخطاف
func (hf *HookFunc) Priority() int {
	return hf.priority
}

// NewAgentHookSystem ينشئ نظام خطافات جديد
func NewAgentHookSystem(logger *zap.Logger) *AgentHookSystem {
	return &AgentHookSystem{
		hooks:   make(map[string][]Hook),
		logger:  logger,
		enabled: true,
	}
}

// RegisterHook يسجل خطاف لنوع حدث معين
func (ahs *AgentHookSystem) RegisterHook(eventType string, hook Hook) {
	ahs.mu.Lock()
	defer ahs.mu.Unlock()

	ahs.hooks[eventType] = append(ahs.hooks[eventType], hook)

	// ترتيب الخطافات حسب الأولوية
	ahs.sortHooks(eventType)

	ahs.logger.Debug("Hook registered",
		zap.String("event_type", eventType),
		zap.String("hook_name", hook.Name()),
		zap.Int("priority", hook.Priority()),
	)
}

// UnregisterHook يلغي تسجيل خطاف
func (ahs *AgentHookSystem) UnregisterHook(eventType string, hookName string) {
	ahs.mu.Lock()
	defer ahs.mu.Unlock()

	hooks := ahs.hooks[eventType]
	for i, hook := range hooks {
		if hook.Name() == hookName {
			// إزالة الخطاف من القائمة
			ahs.hooks[eventType] = append(hooks[:i], hooks[i+1:]...)
			break
		}
	}

	ahs.logger.Debug("Hook unregistered",
		zap.String("event_type", eventType),
		zap.String("hook_name", hookName),
	)
}

// TriggerHook يطلق خطافات لنوع حدث معين
func (ahs *AgentHookSystem) TriggerHook(ctx context.Context, eventType string, event HookEvent) error {
	if !ahs.enabled {
		return nil
	}

	ahs.mu.RLock()
	hooks := ahs.hooks[eventType]
	ahs.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	ahs.logger.Debug("Triggering hooks",
		zap.String("event_type", eventType),
		zap.Int("hook_count", len(hooks)),
	)

	var lastErr error
	for _, hook := range hooks {
		if err := hook.Execute(ctx, event); err != nil {
			ahs.logger.Error("Hook execution failed",
				zap.String("event_type", eventType),
				zap.String("hook_name", hook.Name()),
				zap.Error(err),
			)
			lastErr = err
		}
	}

	return lastErr
}

// sortHooks يرتب الخطافات حسب الأولوية
func (ahs *AgentHookSystem) sortHooks(eventType string) {
	hooks := ahs.hooks[eventType]

	// فرز بسيط حسب الأولوية (الأولوية الأعلى أولاً)
	for i := 0; i < len(hooks)-1; i++ {
		for j := i + 1; j < len(hooks); j++ {
			if hooks[i].Priority() < hooks[j].Priority() {
				hooks[i], hooks[j] = hooks[j], hooks[i]
			}
		}
	}
}

// Enable يفعّل نظام الخطافات
func (ahs *AgentHookSystem) Enable() {
	ahs.mu.Lock()
	defer ahs.mu.Unlock()

	ahs.enabled = true
	ahs.logger.Info("Hook system enabled")
}

// Disable يعطل نظام الخطافات
func (ahs *AgentHookSystem) Disable() {
	ahs.mu.Lock()
	defer ahs.mu.Unlock()

	ahs.enabled = false
	ahs.logger.Info("Hook system disabled")
}

// IsEnabled يرجع حالة التفعيل
func (ahs *AgentHookSystem) IsEnabled() bool {
	ahs.mu.RLock()
	defer ahs.mu.RUnlock()

	return ahs.enabled
}

// GetHookCount يرجع عدد الخطافات لنوع حدث معين
func (ahs *AgentHookSystem) GetHookCount(eventType string) int {
	ahs.mu.RLock()
	defer ahs.mu.RUnlock()

	return len(ahs.hooks[eventType])
}

// GetHookInfo يرجع معلومات عن الخطافات
func (ahs *AgentHookSystem) GetHookInfo() map[string]interface{} {
	ahs.mu.RLock()
	defer ahs.mu.RUnlock()

	info := make(map[string]interface{})
	for eventType, hooks := range ahs.hooks {
		hookNames := make([]string, 0, len(hooks))
		for _, hook := range hooks {
			hookNames = append(hookNames, hook.Name())
		}
		info[eventType] = hookNames
	}

	info["enabled"] = ahs.enabled
	info["total_hooks"] = len(ahs.hooks)

	return info
}

// ClearHooks يمسح جميع الخطافات
func (ahs *AgentHookSystem) ClearHooks() {
	ahs.mu.Lock()
	defer ahs.mu.Unlock()

	ahs.hooks = make(map[string][]Hook)

	ahs.logger.Info("All hooks cleared")
}

// ClearHooksForEvent يمسح الخطافات لنوع حدث معين
func (ahs *AgentHookSystem) ClearHooksForEvent(eventType string) {
	ahs.mu.Lock()
	defer ahs.mu.Unlock()

	delete(ahs.hooks, eventType)

	ahs.logger.Debug("Hooks cleared for event",
		zap.String("event_type", eventType),
	)
}
