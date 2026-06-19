package core

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AnalyticsManager مدير التحليلات
type AnalyticsManager struct {
	events   map[string]*EventMetrics
	sessions map[string]*SessionMetrics
	agents   map[string]*AgentMetrics
	logger   *zap.Logger
	mu       sync.RWMutex
	storage  AnalyticsStorage
	eventBus EventBus
}

// AnalyticsStorage واجهة تخزين التحليلات
type AnalyticsStorage interface {
	StoreEvent(event *EventRecord) error
	StoreSession(session *SessionMetrics) error
	StoreAgent(agent *AgentMetrics) error
	GetEvents(filter EventFilter) ([]*EventRecord, error)
	GetSessions(filter SessionFilter) ([]*SessionMetrics, error)
	GetAgents(filter AgentFilter) ([]*AgentMetrics, error)
}

// EventBus واجهة ناقل الأحداث
type EventBus interface {
	Publish(event string, data interface{}) error
	Subscribe(event string, handler func(data interface{})) error
}

// EventRecord سجل الحدث
type EventRecord struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"session_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
}

// EventMetrics مقاييس الحدث
type EventMetrics struct {
	EventType   string    `json:"event_type"`
	Count       int64     `json:"count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	SuccessRate float64   `json:"success_rate"`
	AvgDuration float64   `json:"avg_duration_ms"`
}

// SessionMetrics مقاييس الجلسة
type SessionMetrics struct {
	SessionID    string                 `json:"session_id"`
	Name         string                 `json:"name"`
	OwnerDID     string                 `json:"owner_did"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Duration     time.Duration          `json:"duration"`
	MessageCount int64                  `json:"message_count"`
	TaskCount    int64                  `json:"task_count"`
	AgentCount   int                    `json:"agent_count"`
	TokensUsed   int64                  `json:"tokens_used"`
	Cost         float64                `json:"cost"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// AgentMetrics مقاييس الوكيل
type AgentMetrics struct {
	AgentID         string                 `json:"agent_id"`
	Name            string                 `json:"name"`
	Provider        string                 `json:"provider"`
	Model           string                 `json:"model"`
	FirstSeen       time.Time              `json:"first_seen"`
	LastSeen        time.Time              `json:"last_seen"`
	TotalTasks      int64                  `json:"total_tasks"`
	CompletedTasks  int64                  `json:"completed_tasks"`
	FailedTasks     int64                  `json:"failed_tasks"`
	SuccessRate     float64                `json:"success_rate"`
	TotalTokens     int64                  `json:"total_tokens"`
	TotalDuration   time.Duration          `json:"total_duration"`
	AvgResponseTime float64                `json:"avg_response_time_ms"`
	Cost            float64                `json:"cost"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// EventFilter فلتر الأحداث
type EventFilter struct {
	EventType string
	SessionID string
	AgentID   string
	UserID    string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
}

// SessionFilter فلتر الجلسات
type SessionFilter struct {
	SessionID string
	OwnerDID  string
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
}

// AgentFilter فلتر الوكلاء
type AgentFilter struct {
	AgentID   string
	Provider  string
	Model     string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
}

// NewAnalyticsManager ينشئ مدير تحليلات جديد
func NewAnalyticsManager(logger *zap.Logger, storage AnalyticsStorage, eventBus EventBus) *AnalyticsManager {
	return &AnalyticsManager{
		events:   make(map[string]*EventMetrics),
		sessions: make(map[string]*SessionMetrics),
		agents:   make(map[string]*AgentMetrics),
		logger:   logger,
		storage:  storage,
		eventBus: eventBus,
	}
}

// RecordEvent يسجل حدث
func (am *AnalyticsManager) RecordEvent(event *EventRecord) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// تحديث مقاييس الحدث
	eventType := event.Type
	if _, exists := am.events[eventType]; !exists {
		am.events[eventType] = &EventMetrics{
			EventType: eventType,
			Count:     0,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
		}
	}

	metrics := am.events[eventType]
	metrics.Count++
	metrics.LastSeen = time.Now()

	// تخزين الحدث
	if am.storage != nil {
		if err := am.storage.StoreEvent(event); err != nil {
			am.logger.Error("فشل تخزين الحدث",
				zap.String("event_type", eventType),
				zap.Error(err))
		}
	}

	am.logger.Debug("تم تسجيل حدث",
		zap.String("event_type", eventType),
		zap.String("event_id", event.ID))

	return nil
}

// UpdateSessionMetrics يحدث مقاييس الجلسة
func (am *AnalyticsManager) UpdateSessionMetrics(sessionID string, updateFunc func(*SessionMetrics)) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	session, exists := am.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	updateFunc(session)
	session.UpdatedAt = time.Now()

	// تخزين مقاييس الجلسة
	if am.storage != nil {
		if err := am.storage.StoreSession(session); err != nil {
			am.logger.Error("فشل تخزين مقاييس الجلسة",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	return nil
}

// UpdateAgentMetrics يحدث مقاييس الوكيل
func (am *AnalyticsManager) UpdateAgentMetrics(agentID string, updateFunc func(*AgentMetrics)) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	agent, exists := am.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	updateFunc(agent)
	agent.LastSeen = time.Now()

	// حساب معدل النجاح
	if agent.TotalTasks > 0 {
		agent.SuccessRate = float64(agent.CompletedTasks) / float64(agent.TotalTasks)
	}

	// حساب متوسط وقت الاستجابة
	if agent.CompletedTasks > 0 {
		agent.AvgResponseTime = float64(agent.TotalDuration.Milliseconds()) / float64(agent.CompletedTasks)
	}

	// تخزين مقاييس الوكيل
	if am.storage != nil {
		if err := am.storage.StoreAgent(agent); err != nil {
			am.logger.Error("فشل تخزين مقاييس الوكيل",
				zap.String("agent_id", agentID),
				zap.Error(err))
		}
	}

	return nil
}

// RegisterSession يسجل جلسة جديدة
func (am *AnalyticsManager) RegisterSession(sessionID, name, ownerDID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.sessions[sessionID]; exists {
		return fmt.Errorf("session already registered: %s", sessionID)
	}

	am.sessions[sessionID] = &SessionMetrics{
		SessionID:    sessionID,
		Name:         name,
		OwnerDID:     ownerDID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Duration:     0,
		MessageCount: 0,
		TaskCount:    0,
		AgentCount:   0,
		TokensUsed:   0,
		Cost:         0,
		Status:       "active",
		Metadata:     make(map[string]interface{}),
	}

	am.logger.Info("تم تسجيل جلسة جديدة للتحليلات",
		zap.String("session_id", sessionID),
		zap.String("name", name),
		zap.String("owner_did", ownerDID))

	return nil
}

// RegisterAgent يسجل وكيل جديد
func (am *AnalyticsManager) RegisterAgent(agentID, name, provider, model string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.agents[agentID]; exists {
		return fmt.Errorf("agent already registered: %s", agentID)
	}

	now := time.Now()
	am.agents[agentID] = &AgentMetrics{
		AgentID:         agentID,
		Name:            name,
		Provider:        provider,
		Model:           model,
		FirstSeen:       now,
		LastSeen:        now,
		TotalTasks:      0,
		CompletedTasks:  0,
		FailedTasks:     0,
		SuccessRate:     0,
		TotalTokens:     0,
		TotalDuration:   0,
		AvgResponseTime: 0,
		Cost:            0,
		Metadata:        make(map[string]interface{}),
	}

	am.logger.Info("تم تسجيل وكيل جديد للتحليلات",
		zap.String("agent_id", agentID),
		zap.String("name", name),
		zap.String("provider", provider),
		zap.String("model", model))

	return nil
}

// GetSessionMetrics يحصل على مقاييس جلسة
func (am *AnalyticsManager) GetSessionMetrics(sessionID string) (*SessionMetrics, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	session, exists := am.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// GetAgentMetrics يحصل على مقاييس وكيل
func (am *AnalyticsManager) GetAgentMetrics(agentID string) (*AgentMetrics, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	agent, exists := am.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

// GetEventMetrics يحصل على مقاييس حدث
func (am *AnalyticsManager) GetEventMetrics(eventType string) (*EventMetrics, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	metrics, exists := am.events[eventType]
	if !exists {
		return nil, fmt.Errorf("event type not found: %s", eventType)
	}

	return metrics, nil
}

// GetAllSessionMetrics يحصل على مقاييس جميع الجلسات
func (am *AnalyticsManager) GetAllSessionMetrics() []*SessionMetrics {
	am.mu.RLock()
	defer am.mu.RUnlock()

	sessions := make([]*SessionMetrics, 0, len(am.sessions))
	for _, session := range am.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// GetAllAgentMetrics يحصل على مقاييس جميع الوكلاء
func (am *AnalyticsManager) GetAllAgentMetrics() []*AgentMetrics {
	am.mu.RLock()
	defer am.mu.RUnlock()

	agents := make([]*AgentMetrics, 0, len(am.agents))
	for _, agent := range am.agents {
		agents = append(agents, agent)
	}

	return agents
}

// GetSummary يحصل على ملخص التحليلات
func (am *AnalyticsManager) GetSummary() map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	totalSessions := len(am.sessions)
	totalAgents := len(am.agents)
	totalEvents := int64(0)

	for _, metrics := range am.events {
		totalEvents += metrics.Count
	}

	totalTokens := int64(0)
	totalCost := 0.0

	for _, agent := range am.agents {
		totalTokens += agent.TotalTokens
		totalCost += agent.Cost
	}

	return map[string]interface{}{
		"total_sessions": totalSessions,
		"total_agents":   totalAgents,
		"total_events":   totalEvents,
		"total_tokens":   totalTokens,
		"total_cost":     totalCost,
	}
}
