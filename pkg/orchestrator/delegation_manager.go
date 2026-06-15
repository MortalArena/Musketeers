package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// DelegationManager مدير التفويضات - يدير تفويض المهام بين الوكلاء
type DelegationManager struct {
	SessionID     string
	AgentRegistry *agent.AgentRegistry
	SessionMgr    *SessionManager
	EventBus      *eventbus.EventBus
	Logger        *zap.Logger

	// بيانات التفويضات
	Delegations map[string]*Delegation // delegationID -> delegation

	mu sync.RWMutex
}

// Delegation تفويض
type Delegation struct {
	ID             string
	SessionID      string
	FromAgentID    string
	ToAgentID      string
	TaskID         string
	DelegationType string // direct, hierarchical, peer
	Permissions    []string
	Constraints    []string
	Status         string // pending, active, completed, revoked
	CreatedAt      time.Time
	ExpiresAt      *time.Time
	CompletedAt    *time.Time
}

// NewDelegationManager ينشئ مدير تفويضات
func NewDelegationManager(sessionID string, logger *zap.Logger) *DelegationManager {
	return &DelegationManager{
		SessionID:   sessionID,
		Logger:      logger,
		Delegations: make(map[string]*Delegation),
	}
}

// SetAgentRegistry يضبط سجل الوكلاء
func (dm *DelegationManager) SetAgentRegistry(registry *agent.AgentRegistry) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.AgentRegistry = registry
}

// SetSessionManager يضبط مدير الجلسة
func (dm *DelegationManager) SetSessionManager(sm *SessionManager) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.SessionMgr = sm
}

// SetEventBus يضبط event bus
func (dm *DelegationManager) SetEventBus(eb *eventbus.EventBus) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.EventBus = eb
}

// CreateDelegation ينشئ تفويضاً جديداً
func (dm *DelegationManager) CreateDelegation(ctx context.Context, fromAgentID, toAgentID, taskID, delegationType string, permissions []string) (*Delegation, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	delegation := &Delegation{
		ID:             fmt.Sprintf("del_%d", time.Now().UnixNano()),
		SessionID:      dm.SessionID,
		FromAgentID:    fromAgentID,
		ToAgentID:      toAgentID,
		TaskID:         taskID,
		DelegationType: delegationType,
		Permissions:    permissions,
		Constraints:    []string{},
		Status:         "active",
		CreatedAt:      time.Now(),
	}

	dm.Delegations[delegation.ID] = delegation

	dm.Logger.Info("تم إنشاء تفويض",
		zap.String("delegation_id", delegation.ID),
		zap.String("from_agent", fromAgentID),
		zap.String("to_agent", toAgentID),
		zap.String("task_id", taskID),
		zap.String("type", delegationType),
	)

	if dm.EventBus != nil {
		dm.EventBus.Publish(eventbus.Event{
			Type:      "delegation.created",
			Payload:   delegation,
			Source:    "delegation_manager",
			SessionID: dm.SessionID,
		})
	}

	return delegation, nil
}

// RevokeDelegation يلغي تفويضاً
func (dm *DelegationManager) RevokeDelegation(ctx context.Context, delegationID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	delegation, exists := dm.Delegations[delegationID]
	if !exists {
		return fmt.Errorf("التفويض %s غير موجود", delegationID)
	}

	if delegation.Status != "active" {
		return fmt.Errorf("التفويض %s غير نشط", delegationID)
	}

	delegation.Status = "revoked"

	dm.Logger.Info("تم إلغاء التفويض",
		zap.String("delegation_id", delegationID),
	)

	if dm.EventBus != nil {
		dm.EventBus.Publish(eventbus.Event{
			Type:      "delegation.revoked",
			Payload:   delegationID,
			Source:    "delegation_manager",
			SessionID: dm.SessionID,
		})
	}

	return nil
}

// CompleteDelegation يكمل تفويضاً
func (dm *DelegationManager) CompleteDelegation(ctx context.Context, delegationID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	delegation, exists := dm.Delegations[delegationID]
	if !exists {
		return fmt.Errorf("التفويض %s غير موجود", delegationID)
	}

	if delegation.Status != "active" {
		return fmt.Errorf("التفويض %s غير نشط", delegationID)
	}

	delegation.Status = "completed"
	now := time.Now()
	delegation.CompletedAt = &now

	dm.Logger.Info("تم إكمال التفويض",
		zap.String("delegation_id", delegationID),
	)

	if dm.EventBus != nil {
		dm.EventBus.Publish(eventbus.Event{
			Type:      "delegation.completed",
			Payload:   delegationID,
			Source:    "delegation_manager",
			SessionID: dm.SessionID,
		})
	}

	return nil
}

// GetDelegation يحصل على تفويض
func (dm *DelegationManager) GetDelegation(delegationID string) (*Delegation, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	delegation, exists := dm.Delegations[delegationID]
	if !exists {
		return nil, fmt.Errorf("التفويض %s غير موجود", delegationID)
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	delegationCopy := *delegation
	return &delegationCopy, nil
}

// GetDelegationsByTask يحصل على تفويضات مهمة
func (dm *DelegationManager) GetDelegationsByTask(taskID string) []*Delegation {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var result []*Delegation
	for _, delegation := range dm.Delegations {
		if delegation.TaskID == taskID {
			delegationCopy := *delegation
			result = append(result, &delegationCopy)
		}
	}

	return result
}

// GetDelegationsByAgent يحصل على تفويضات وكيل
func (dm *DelegationManager) GetDelegationsByAgent(agentID string) []*Delegation {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var result []*Delegation
	for _, delegation := range dm.Delegations {
		if delegation.FromAgentID == agentID || delegation.ToAgentID == agentID {
			delegationCopy := *delegation
			result = append(result, &delegationCopy)
		}
	}

	return result
}

// CheckPermission يتحقق من صلاحية
func (dm *DelegationManager) CheckPermission(agentID, taskID, permission string) bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, delegation := range dm.Delegations {
		if delegation.TaskID == taskID && delegation.ToAgentID == agentID && delegation.Status == "active" {
			for _, perm := range delegation.Permissions {
				if perm == permission {
					return true
				}
			}
		}
	}

	return false
}

// CleanupExpired ينظف التفويضات المنتهية
func (dm *DelegationManager) CleanupExpired(ctx context.Context) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	now := time.Now()
	cleanedCount := 0

	for _, delegation := range dm.Delegations {
		if delegation.ExpiresAt != nil && delegation.ExpiresAt.Before(now) {
			if delegation.Status == "active" {
				delegation.Status = "revoked"
				cleanedCount++
			}
		}
	}

	dm.Logger.Info("تم تنظيف التفويضات المنتهية",
		zap.Int("cleaned_count", cleanedCount),
	)

	return nil
}
