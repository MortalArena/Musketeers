package orchestrator

import (
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"go.uber.org/zap"
)

// AgentRole دور الوكيل
type AgentRole string

const (
	RoleLeader      AgentRole = "leader"      // قائد الفريق
	RoleReviewer    AgentRole = "reviewer"    // مراجع
	RoleExecutor    AgentRole = "executor"    // منفذ
	RoleTester      AgentRole = "tester"      // مختبر
	RoleDocumenter  AgentRole = "documenter"  // موثق
	RoleAnalyst     AgentRole = "analyst"     // محلل
	RoleDesigner    AgentRole = "designer"    // مصمم
	RoleCoordinator AgentRole = "coordinator" // منسق
)

// RoleAssignment تعيين الدور
type RoleAssignment struct {
	AgentID      string                  `json:"agent_id"`
	Role         AgentRole               `json:"role"`
	Weight       float64                 `json:"weight"`
	Capabilities []agent.AgentCapability `json:"capabilities"`
	AssignedAt   int64                   `json:"assigned_at"`
}

// RoleAssigner مدير تعيين الأدوار
type RoleAssigner struct {
	registry *agent.AgentRegistry
	roles    map[string][]*RoleAssignment // role -> assignments
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewRoleAssigner ينشئ مدير تعيين أدوار جديد
func NewRoleAssigner(registry *agent.AgentRegistry) *RoleAssigner {
	return &RoleAssigner{
		registry: registry,
		roles:    make(map[string][]*RoleAssignment),
		logger:   zap.NewNop(),
	}
}

// SetLogger يضبط logger
func (ra *RoleAssigner) SetLogger(logger *zap.Logger) {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	ra.logger = logger
}

// AssignRole يعين دوراً لوكيل
func (ra *RoleAssigner) AssignRole(agentID string, role AgentRole, weight float64) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	// التحقق من وجود الوكيل
	agent, err := ra.registry.Get(agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	// الحصول على قدرات الوكيل
	capabilities := agent.GetCapabilities()

	// إنشاء تعيين الدور
	assignment := &RoleAssignment{
		AgentID:      agentID,
		Role:         role,
		Weight:       weight,
		Capabilities: capabilities,
		AssignedAt:   0, // سيتم تعيينه لاحقاً
	}

	// إضافة التعيين إلى القائمة
	roleKey := string(role)
	ra.roles[roleKey] = append(ra.roles[roleKey], assignment)

	ra.logger.Info("Role assigned",
		zap.String("agent_id", agentID),
		zap.String("role", string(role)),
		zap.Float64("weight", weight),
	)

	return nil
}

// UnassignRole يلغي تعيين دور من وكيل
func (ra *RoleAssigner) UnassignRole(agentID string, role AgentRole) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	roleKey := string(role)
	assignments, exists := ra.roles[roleKey]
	if !exists {
		return fmt.Errorf("role %s not found", role)
	}

	// إزالة التعيين
	newAssignments := make([]*RoleAssignment, 0)
	for _, assignment := range assignments {
		if assignment.AgentID != agentID {
			newAssignments = append(newAssignments, assignment)
		}
	}

	ra.roles[roleKey] = newAssignments

	ra.logger.Info("Role unassigned",
		zap.String("agent_id", agentID),
		zap.String("role", string(role)),
	)

	return nil
}

// GetAgentsByRole يحصل على الوكلاء حسب الدور
func (ra *RoleAssigner) GetAgentsByRole(role AgentRole) []*RoleAssignment {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	roleKey := string(role)
	assignments, exists := ra.roles[roleKey]
	if !exists {
		return []*RoleAssignment{}
	}

	// إنشاء نسخة لتجنب التعديل الخارجي
	result := make([]*RoleAssignment, len(assignments))
	copy(result, assignments)

	return result
}

// GetRolesByAgent يحصل على أدوار وكيل
func (ra *RoleAssigner) GetRolesByAgent(agentID string) []AgentRole {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	var roles []AgentRole
	for roleKey, assignments := range ra.roles {
		for _, assignment := range assignments {
			if assignment.AgentID == agentID {
				roles = append(roles, AgentRole(roleKey))
				break
			}
		}
	}

	return roles
}

// GetBestAgentForRole يحصل على أفضل وكيل لدور معين
func (ra *RoleAssigner) GetBestAgentForRole(role AgentRole, requiredCapabilities []agent.AgentCapability) (string, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	roleKey := string(role)
	assignments, exists := ra.roles[roleKey]
	if !exists {
		return "", fmt.Errorf("no agents assigned to role %s", role)
	}

	// البحث عن أفضل وكيل بناءً على الوزن والقدرات
	var bestAgent string
	bestScore := 0.0

	for _, assignment := range assignments {
		score := assignment.Weight

		// التحقق من القدرات المطلوبة
		if len(requiredCapabilities) > 0 {
			capabilityMatch := 0
			for _, required := range requiredCapabilities {
				for _, cap := range assignment.Capabilities {
					if cap == required {
						capabilityMatch++
						break
					}
				}
			}
			score += float64(capabilityMatch) / float64(len(requiredCapabilities)) * 0.5
		}

		if score > bestScore {
			bestScore = score
			bestAgent = assignment.AgentID
		}
	}

	if bestAgent == "" {
		return "", fmt.Errorf("no suitable agent found for role %s", role)
	}

	return bestAgent, nil
}

// GetAllRoles يحصل على جميع الأدوار المعينة
func (ra *RoleAssigner) GetAllRoles() map[string][]*RoleAssignment {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	result := make(map[string][]*RoleAssignment, len(ra.roles))
	for k, v := range ra.roles {
		assignments := make([]*RoleAssignment, len(v))
		copy(assignments, v)
		result[k] = assignments
	}

	return result
}

// ClearRole يمسح جميع التعيينات لدور معين
func (ra *RoleAssigner) ClearRole(role AgentRole) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	roleKey := string(role)
	delete(ra.roles, roleKey)

	ra.logger.Info("Role cleared",
		zap.String("role", string(role)),
	)
}

// ClearAll يمسح جميع التعيينات
func (ra *RoleAssigner) ClearAll() {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	ra.roles = make(map[string][]*RoleAssignment)

	ra.logger.Info("All roles cleared")
}

// GetRoleCount يحصل على عدد الوكلاء لكل دور
func (ra *RoleAssigner) GetRoleCount() map[string]int {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	result := make(map[string]int, len(ra.roles))
	for roleKey, assignments := range ra.roles {
		result[roleKey] = len(assignments)
	}

	return result
}

// SuggestRole يقترح دوراً مناسباً لوكيل بناءً على قدراته
func (ra *RoleAssigner) SuggestRole(agentID string) (AgentRole, error) {
	agentObj, err := ra.registry.Get(agentID)
	if err != nil {
		return "", fmt.Errorf("agent not found: %w", err)
	}

	capabilities := agentObj.GetCapabilities()

	// تحديد الدور المناسب بناءً على القدرات
	for _, cap := range capabilities {
		switch cap {
		case agent.CapabilityCodeGeneration:
			return RoleExecutor, nil
		case agent.CapabilityCodeReview:
			return RoleReviewer, nil
		case agent.CapabilityTesting:
			return RoleTester, nil
		case agent.CapabilityDocumentation:
			return RoleDocumenter, nil
		case agent.CapabilityAnalysis:
			return RoleAnalyst, nil
		case agent.CapabilityDesign:
			return RoleDesigner, nil
		}
	}

	// إذا لم يتم العثور على دور محدد، نستخدم Coordinator
	return RoleCoordinator, nil
}
