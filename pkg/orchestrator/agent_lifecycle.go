package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// LifecycleState حالة دورة حياة الوكيل
type LifecycleState string

const (
	LifecycleStateIdle      LifecycleState = "idle"
	LifecycleStateStarting  LifecycleState = "starting"
	LifecycleStateRunning   LifecycleState = "running"
	LifecycleStateStopping  LifecycleState = "stopping"
	LifecycleStateStopped   LifecycleState = "stopped"
	LifecycleStateError     LifecycleState = "error"
)

// AgentLifecycleManager مدير دورة حياة الوكيل
type AgentLifecycleManager struct {
	registry *agent.AgentRegistry
	states   map[string]LifecycleState // agentID -> state
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewAgentLifecycleManager ينشئ مدير دورة حياة جديد
func NewAgentLifecycleManager(registry *agent.AgentRegistry) *AgentLifecycleManager {
	return &AgentLifecycleManager{
		registry: registry,
		states:   make(map[string]LifecycleState),
		logger:   zap.NewNop(),
	}
}

// SetLogger يضبط logger
func (alm *AgentLifecycleManager) SetLogger(logger *zap.Logger) {
	alm.mu.Lock()
	defer alm.mu.Unlock()
	alm.logger = logger
}

// StartAgent يبدأ وكيل
func (alm *AgentLifecycleManager) StartAgent(ctx context.Context, agentID string) error {
	alm.mu.Lock()
	defer alm.mu.Unlock()

	// التحقق من الحالة الحالية
	currentState, exists := alm.states[agentID]
	if exists && currentState != LifecycleStateIdle && currentState != LifecycleStateStopped && currentState != LifecycleStateError {
		return fmt.Errorf("agent %s is not in a startable state: %s", agentID, currentState)
	}

	// تحديث الحالة إلى starting
	alm.states[agentID] = LifecycleStateStarting

	alm.logger.Info("Starting agent",
		zap.String("agent_id", agentID),
	)

	// الحصول على الوكيل من السجل
	agent, err := alm.registry.Get(agentID)
	if err != nil {
		alm.states[agentID] = LifecycleStateError
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// التحقق من توفر الوكيل
	if !agent.IsAvailable() {
		alm.states[agentID] = LifecycleStateError
		return fmt.Errorf("agent %s is not available", agentID)
	}

	// تحديث الحالة إلى running
	alm.states[agentID] = LifecycleStateRunning

	alm.logger.Info("Agent started successfully",
		zap.String("agent_id", agentID),
	)

	return nil
}

// StopAgent يوقف وكيل
func (alm *AgentLifecycleManager) StopAgent(ctx context.Context, agentID string) error {
	alm.mu.Lock()
	defer alm.mu.Unlock()

	// التحقق من الحالة الحالية
	currentState, exists := alm.states[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found in lifecycle manager", agentID)
	}

	if currentState == LifecycleStateStopped || currentState == LifecycleStateStopping {
		return fmt.Errorf("agent %s is already stopped or stopping", agentID)
	}

	// تحديث الحالة إلى stopping
	alm.states[agentID] = LifecycleStateStopping

	alm.logger.Info("Stopping agent",
		zap.String("agent_id", agentID),
	)

	// الحصول على الوكيل من السجل
	agent, err := alm.registry.Get(agentID)
	if err != nil {
		alm.states[agentID] = LifecycleStateError
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// إغلاق الوكيل
	if err := agent.Close(); err != nil {
		alm.states[agentID] = LifecycleStateError
		return fmt.Errorf("failed to close agent: %w", err)
	}

	// تحديث الحالة إلى stopped
	alm.states[agentID] = LifecycleStateStopped

	alm.logger.Info("Agent stopped successfully",
		zap.String("agent_id", agentID),
	)

	return nil
}

// RestartAgent يعيد تشغيل وكيل
func (alm *AgentLifecycleManager) RestartAgent(ctx context.Context, agentID string) error {
	alm.logger.Info("Restarting agent",
		zap.String("agent_id", agentID),
	)

	// إيقاف الوكيل
	if err := alm.StopAgent(ctx, agentID); err != nil {
		return fmt.Errorf("failed to stop agent: %w", err)
	}

	// انتظار قصير
	time.Sleep(100 * time.Millisecond)

	// بدء الوكيل
	if err := alm.StartAgent(ctx, agentID); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	alm.logger.Info("Agent restarted successfully",
		zap.String("agent_id", agentID),
	)

	return nil
}

// GetState يحصل على حالة دورة حياة وكيل
func (alm *AgentLifecycleManager) GetState(agentID string) (LifecycleState, error) {
	alm.mu.RLock()
	defer alm.mu.RUnlock()

	state, exists := alm.states[agentID]
	if !exists {
		return "", fmt.Errorf("agent %s not found in lifecycle manager", agentID)
	}

	return state, nil
}

// GetAllStates يحصل على جميع حالات دورة الحياة
func (alm *AgentLifecycleManager) GetAllStates() map[string]LifecycleState {
	alm.mu.RLock()
	defer alm.mu.RUnlock()

	result := make(map[string]LifecycleState, len(alm.states))
	for k, v := range alm.states {
		result[k] = v
	}

	return result
}

// GetRunningAgents يحصل على الوكلاء الجارية
func (alm *AgentLifecycleManager) GetRunningAgents() []string {
	alm.mu.RLock()
	defer alm.mu.RUnlock()

	var running []string
	for agentID, state := range alm.states {
		if state == LifecycleStateRunning {
			running = append(running, agentID)
		}
	}

	return running
}

// GetIdleAgents يحصل على الوكلاء الخاملة
func (alm *AgentLifecycleManager) GetIdleAgents() []string {
	alm.mu.RLock()
	defer alm.mu.RUnlock()

	var idle []string
	for agentID, state := range alm.states {
		if state == LifecycleStateIdle || state == LifecycleStateStopped {
			idle = append(idle, agentID)
		}
	}

	return idle
}

// GetErrorAgents يحصل على الوكلاء في حالة خطأ
func (alm *AgentLifecycleManager) GetErrorAgents() []string {
	alm.mu.RLock()
	defer alm.mu.RUnlock()

	var errorAgents []string
	for agentID, state := range alm.states {
		if state == LifecycleStateError {
			errorAgents = append(errorAgents, agentID)
		}
	}

	return errorAgents
}

// InitializeAgent يهيئ وكيل في السجل
func (alm *AgentLifecycleManager) InitializeAgent(agentID string) {
	alm.mu.Lock()
	defer alm.mu.Unlock()

	// تعيين الحالة الافتراضية
	alm.states[agentID] = LifecycleStateIdle

	alm.logger.Info("Agent initialized in lifecycle manager",
		zap.String("agent_id", agentID),
	)
}

// RemoveAgent يزيل وكيل من مدير دورة الحياة
func (alm *AgentLifecycleManager) RemoveAgent(agentID string) {
	alm.mu.Lock()
	defer alm.mu.Unlock()

	delete(alm.states, agentID)

	alm.logger.Info("Agent removed from lifecycle manager",
		zap.String("agent_id", agentID),
	)
}

// HealthCheck فحص صحة الوكلاء
func (alm *AgentLifecycleManager) HealthCheck(ctx context.Context) map[string]bool {
	alm.mu.RLock()
	defer alm.mu.RUnlock()

	results := make(map[string]bool)

	for agentID := range alm.states {
		agent, err := alm.registry.Get(agentID)
		if err != nil {
			results[agentID] = false
			continue
		}

		results[agentID] = agent.IsAvailable()
	}

	return results
}

// GetStats يحصل على إحصائيات دورة الحياة
func (alm *AgentLifecycleManager) GetStats() map[string]interface{} {
	alm.mu.RLock()
	defer alm.mu.RUnlock()

	running := 0
	idle := 0
	stopped := 0
	errorCount := 0
	starting := 0
	stopping := 0

	for _, state := range alm.states {
		switch state {
		case LifecycleStateRunning:
			running++
		case LifecycleStateIdle:
			idle++
		case LifecycleStateStopped:
			stopped++
		case LifecycleStateError:
			errorCount++
		case LifecycleStateStarting:
			starting++
		case LifecycleStateStopping:
			stopping++
		}
	}

	return map[string]interface{}{
		"total":    len(alm.states),
		"running":  running,
		"idle":     idle,
		"stopped":  stopped,
		"error":    errorCount,
		"starting": starting,
		"stopping": stopping,
	}
}
