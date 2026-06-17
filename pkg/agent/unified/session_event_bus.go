package unified

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SessionEventBus ناقل أحداث الجلسة لمزامنة لحظية
type SessionEventBus struct {
	sessionID string
	logger    *zap.Logger
	mu        sync.RWMutex

	// قنوات الأحداث
	eventQueue       chan *SessionEvent
	agentSubscribers map[string]chan *SessionEvent
	sessionManager   chan *SessionEvent

	// حالة الأحداث
	eventHistory  []*SessionEvent
	active        bool
	lastEventTime time.Time
	totalEvents   int
}

// SessionEvent حدث في الجلسة
type SessionEvent struct {
	ID          string
	SessionID   string
	SourceAgent string
	TargetAgent string // فارغ يعني جميع الوكلاء
	EventType   SessionEventType
	Timestamp   time.Time
	Priority    EventPriority
	Data        interface{}
	Metadata    map[string]interface{}
}

// SessionEventType نوع حدث الجلسة
type SessionEventType string

const (
	// أحداث المهام
	TaskStarted   SessionEventType = "task_started"
	TaskProgress  SessionEventType = "task_progress"
	TaskCompleted SessionEventType = "task_completed"
	TaskFailed    SessionEventType = "task_failed"
	TaskAssigned  SessionEventType = "task_assigned"

	// أحداث الذاكرة
	MemoryUpdated  SessionEventType = "memory_updated"
	MemoryAccessed SessionEventType = "memory_accessed"
	MemoryCreated  SessionEventType = "memory_created"

	// أحداث المهارات
	SkillLearned  SessionEventType = "skill_learned"
	SkillImproved SessionEventType = "skill_improved"
	SkillUsed     SessionEventType = "skill_used"

	// أحداث التواصل
	AgentMessage       SessionEventType = "agent_message"
	AgentStatus        SessionEventType = "agent_status"
	SessionStatusEvent SessionEventType = "session_status"

	// أحداث النظام
	SystemAlert SessionEventType = "system_alert"
	SystemError SessionEventType = "system_error"
)

// EventPriority أولوية الحدث
type EventPriority string

const (
	PriorityLow      EventPriority = "low"
	PriorityMedium   EventPriority = "medium"
	PriorityHigh     EventPriority = "high"
	PriorityCritical EventPriority = "critical"
)

// NewSessionEventBus ينشئ ناقل أحداث جلسة جديد
func NewSessionEventBus(sessionID string, logger *zap.Logger) *SessionEventBus {
	return &SessionEventBus{
		sessionID:        sessionID,
		logger:           logger,
		eventQueue:       make(chan *SessionEvent, 1000),
		agentSubscribers: make(map[string]chan *SessionEvent),
		sessionManager:   make(chan *SessionEvent, 1000),
		eventHistory:     []*SessionEvent{},
		active:           true,
		lastEventTime:    time.Now(),
		totalEvents:      0,
	}
}

// Start يبدأ ناقل الأحداث
func (seb *SessionEventBus) Start(ctx context.Context) {
	seb.mu.Lock()
	seb.active = true
	seb.mu.Unlock()

	go seb.processEvents(ctx)
}

// Stop يوقف ناقل الأحداث
func (seb *SessionEventBus) Stop() {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	seb.active = false
	close(seb.eventQueue)
	close(seb.sessionManager)

	// إغلاق جميع قنوات الوكلاء
	for _, ch := range seb.agentSubscribers {
		close(ch)
	}
}

// processEvents يعالج الأحداث
func (seb *SessionEventBus) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			seb.logger.Info("تم إيقاف معالجة أحداث الجلسة")
			return
		case event, ok := <-seb.eventQueue:
			if !ok {
				seb.logger.Info("تم إغلاق قناة أحداث الجلسة")
				return
			}
			seb.distributeEvent(event)
		}
	}
}

// distributeEvent يوزع الحدث على المشتركين
func (seb *SessionEventBus) distributeEvent(event *SessionEvent) {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	// إضافة إلى التاريخ
	seb.eventHistory = append(seb.eventHistory, event)
	seb.lastEventTime = event.Timestamp
	seb.totalEvents++

	// إرسال لمدير الجلسة دائماً
	select {
	case seb.sessionManager <- event:
	default:
		seb.logger.Warn("قناة مدير الجلسة ممتلئة")
	}

	// إرسال للوكيل المستهدف أو جميع الوكلاء
	if event.TargetAgent == "" {
		// إرسال لجميع الوكلاء
		for agentID, ch := range seb.agentSubscribers {
			select {
			case ch <- event:
			default:
				seb.logger.Warn("قناة الوكيل ممتلئة", zap.String("agent_id", agentID))
			}
		}
	} else {
		// إرسال للوكيل المستهدف
		if ch, exists := seb.agentSubscribers[event.TargetAgent]; exists {
			select {
			case ch <- event:
			default:
				seb.logger.Warn("قناة الوكيل المستهدف ممتلئة", zap.String("agent_id", event.TargetAgent))
			}
		}
	}

	seb.logger.Debug("تم توزيع الحدث",
		zap.String("session_id", seb.sessionID),
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.EventType)),
		zap.String("source_agent", event.SourceAgent),
		zap.String("target_agent", event.TargetAgent))
}

// PublishEvent ينشر حدث
func (seb *SessionEventBus) PublishEvent(ctx context.Context, event *SessionEvent) error {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	if !seb.active {
		return nil
	}

	// إرسال الحدث
	select {
	case seb.eventQueue <- event:
		seb.logger.Info("تم نشر الحدث",
			zap.String("session_id", seb.sessionID),
			zap.String("event_id", event.ID),
			zap.String("event_type", string(event.EventType)),
			zap.String("source_agent", event.SourceAgent))
		return nil
	default:
		seb.logger.Warn("قناة الأحداث ممتلئة")
		return nil
	}
}

// SubscribeAgent يربط وكيل بناقل الأحداث
func (seb *SessionEventBus) SubscribeAgent(agentID string) chan *SessionEvent {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	ch := make(chan *SessionEvent, 100)
	seb.agentSubscribers[agentID] = ch

	seb.logger.Info("تم اشتراك الوكيل في ناقل الأحداث",
		zap.String("session_id", seb.sessionID),
		zap.String("agent_id", agentID))

	return ch
}

// UnsubscribeAgent يفصل وكيل من ناقل الأحداث
func (seb *SessionEventBus) UnsubscribeAgent(agentID string) {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	if ch, exists := seb.agentSubscribers[agentID]; exists {
		close(ch)
		delete(seb.agentSubscribers, agentID)
	}

	seb.logger.Info("تم فصل الوكيل من ناقل الأحداث",
		zap.String("session_id", seb.sessionID),
		zap.String("agent_id", agentID))
}

// GetSessionManagerChannel يحصل على قناة مدير الجلسة
func (seb *SessionEventBus) GetSessionManagerChannel() chan *SessionEvent {
	return seb.sessionManager
}

// GetAgentChannel يحصل على قناة وكيل
func (seb *SessionEventBus) GetAgentChannel(agentID string) (chan *SessionEvent, bool) {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	ch, exists := seb.agentSubscribers[agentID]
	return ch, exists
}

// GetEventHistory يحصل على تاريخ الأحداث
func (seb *SessionEventBus) GetEventHistory(limit int) []*SessionEvent {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	if limit <= 0 || limit > len(seb.eventHistory) {
		limit = len(seb.eventHistory)
	}

	start := len(seb.eventHistory) - limit
	if start < 0 {
		start = 0
	}

	return seb.eventHistory[start:]
}

// GetRecentEventsForAgent يحصل على الأحداث الأخيرة لوكيل معين
func (seb *SessionEventBus) GetRecentEventsForAgent(agentID string, limit int) []*SessionEvent {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	var events []*SessionEvent
	for i := len(seb.eventHistory) - 1; i >= 0 && len(events) < limit; i-- {
		event := seb.eventHistory[i]
		// أحداث مرتبطة بالوكيل (مصدر أو مستهدف)
		if event.SourceAgent == agentID || event.TargetAgent == agentID || event.TargetAgent == "" {
			events = append(events, event)
		}
	}

	return events
}

// GetStatus يحصل على حالة ناقل الأحداث
func (seb *SessionEventBus) GetStatus() map[string]interface{} {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	return map[string]interface{}{
		"active":         seb.active,
		"last_event":     seb.lastEventTime,
		"total_events":   seb.totalEvents,
		"subscribers":    len(seb.agentSubscribers),
		"pending_events": len(seb.eventQueue),
		"history_size":   len(seb.eventHistory),
	}
}

// BroadcastToAll يرسل حدث لجميع الوكلاء
func (seb *SessionEventBus) BroadcastToAll(ctx context.Context, sourceAgent string, eventType SessionEventType, data interface{}) error {
	event := &SessionEvent{
		ID:          generateID(),
		SessionID:   seb.sessionID,
		SourceAgent: sourceAgent,
		TargetAgent: "", // فارغ يعني جميع الوكلاء
		EventType:   eventType,
		Timestamp:   time.Now(),
		Priority:    PriorityMedium,
		Data:        data,
		Metadata:    make(map[string]interface{}),
	}

	return seb.PublishEvent(ctx, event)
}

// SendToAgent يرسل حدث لوكيل محدد
func (seb *SessionEventBus) SendToAgent(ctx context.Context, sourceAgent, targetAgent string, eventType SessionEventType, data interface{}) error {
	event := &SessionEvent{
		ID:          generateID(),
		SessionID:   seb.sessionID,
		SourceAgent: sourceAgent,
		TargetAgent: targetAgent,
		EventType:   eventType,
		Timestamp:   time.Now(),
		Priority:    PriorityHigh,
		Data:        data,
		Metadata:    make(map[string]interface{}),
	}

	return seb.PublishEvent(ctx, event)
}

// SendToSessionManager يرسل حدث لمدير الجلسة
func (seb *SessionEventBus) SendToSessionManager(ctx context.Context, sourceAgent string, eventType SessionEventType, data interface{}) error {
	event := &SessionEvent{
		ID:          generateID(),
		SessionID:   seb.sessionID,
		SourceAgent: sourceAgent,
		TargetAgent: "session_manager",
		EventType:   eventType,
		Timestamp:   time.Now(),
		Priority:    PriorityHigh,
		Data:        data,
		Metadata:    make(map[string]interface{}),
	}

	return seb.PublishEvent(ctx, event)
}
