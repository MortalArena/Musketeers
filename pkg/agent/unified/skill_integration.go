package unified

import (
	"context"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent/skills"
	"github.com/MortalArena/Musketeers/pkg/session"
	"go.uber.org/zap"
)

// SkillIntegration تكامل نظام المهارات
type SkillIntegration struct {
	sessionSkills *session.SkillsManager
	skillManager  *skills.SkillManager
	logger        *zap.Logger
	mu            sync.RWMutex
}

// NewSkillIntegration ينشئ تكامل مهارات جديد
func NewSkillIntegration(sessionSkills *session.SkillsManager, skillManager *skills.SkillManager, logger *zap.Logger) *SkillIntegration {
	return &SkillIntegration{
		sessionSkills: sessionSkills,
		skillManager:  skillManager,
		logger:        logger,
	}
}

// Initialize يهيئ تكامل المهارات
func (si *SkillIntegration) Initialize(ctx context.Context) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.logger.Info("تم تهيئة تكامل المهارات")
	return nil
}

// GetSummary يحصل على ملخص تكامل المهارات
func (si *SkillIntegration) GetSummary() map[string]interface{} {
	si.mu.RLock()
	defer si.mu.RUnlock()

	return map[string]interface{}{
		"session_skills_enabled": si.sessionSkills != nil,
		"skill_manager_enabled":  si.skillManager != nil,
		"integrated":             true,
	}
}
