package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// generateLogID يولد معرف فريد للسجل
func generateLogID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ============================================================
// ComprehensiveLogger - نظام تسجيل شامل
// ============================================================

// ComprehensiveLogger يسجل كل شيء ويبثه للوكلاء جميعاً
type ComprehensiveLogger struct {
	// المكونات الأساسية
	eventBus *eventbus.EventBus

	// سجلات الأحداث
	logs map[string]*SystemLog
	mu   sync.RWMutex

	// Channels للتواصل الداخلي
	logToEventBus chan *SystemLog
	eventBusToLog chan eventbus.Event

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Logger
	logger *zap.Logger

	// Metrics
	metrics *LoggerMetrics
}

// LoggerMetrics مقاييس التسجيل
type LoggerMetrics struct {
	LogsRecorded    int64
	LogsBroadcasted int64
	Errors          int64
	LastActivity    time.Time
}

// SystemLog سجل نظام
type SystemLog struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // agent_action, system_event, user_action, error, info
	Source      string                 `json:"source"`
	SessionID   string                 `json:"session_id,omitempty"`
	AgentID     string                 `json:"agent_id,omitempty"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    string                 `json:"severity"` // debug, info, warning, error, critical
}

// NewComprehensiveLogger ينشئ ComprehensiveLogger جديد
func NewComprehensiveLogger(eventBus *eventbus.EventBus, logger *zap.Logger) *ComprehensiveLogger {
	ctx, cancel := context.WithCancel(context.Background())

	return &ComprehensiveLogger{
		eventBus:      eventBus,
		logs:          make(map[string]*SystemLog),
		logToEventBus: make(chan *SystemLog, 1000),
		eventBusToLog: make(chan eventbus.Event, 1000),
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
		metrics:       &LoggerMetrics{},
	}
}

// Start يبدأ ComprehensiveLogger
func (cl *ComprehensiveLogger) Start() error {
	cl.logger.Info("بدء ComprehensiveLogger")

	// الاشتراك في أحداث Event Bus
	cl.subscribeToEventBus()

	// بدء معالج التسجيل
	cl.wg.Add(1)
	go cl.logHandler()

	// بدء معالج Event Bus
	cl.wg.Add(1)
	go cl.eventBusHandler()

	cl.logger.Info("تم بدء ComprehensiveLogger بنجاح")
	return nil
}

// Stop يوقف ComprehensiveLogger
func (cl *ComprehensiveLogger) Stop() error {
	cl.logger.Info("إيقاف ComprehensiveLogger")

	cl.cancel()
	cl.wg.Wait()

	close(cl.logToEventBus)
	close(cl.eventBusToLog)

	cl.logger.Info("تم إيقاف ComprehensiveLogger بنجاح")
	return nil
}

// ============================================================
// تسجيل الأحداث
// ============================================================

// LogAction يسجل إجراء
func (cl *ComprehensiveLogger) LogAction(source, agentID, description string, data map[string]interface{}) {
	log := &SystemLog{
		ID:          generateLogID(),
		Type:        "agent_action",
		Source:      source,
		AgentID:     agentID,
		Description: description,
		Data:        data,
		Timestamp:   time.Now(),
		Severity:    "info",
	}

	cl.recordLog(log)
}

// LogEvent يسجل حدث نظام
func (cl *ComprehensiveLogger) LogEvent(source, description string, data map[string]interface{}) {
	log := &SystemLog{
		ID:          generateLogID(),
		Type:        "system_event",
		Source:      source,
		Description: description,
		Data:        data,
		Timestamp:   time.Now(),
		Severity:    "info",
	}

	cl.recordLog(log)
}

// LogUserAction يسجل إجراء مستخدم
func (cl *ComprehensiveLogger) LogUserAction(userID, description string, data map[string]interface{}) {
	log := &SystemLog{
		ID:          generateChatID(),
		Type:        "user_action",
		Source:      userID,
		Description: description,
		Data:        data,
		Timestamp:   time.Now(),
		Severity:    "info",
	}

	cl.recordLog(log)
}

// LogError يسجل خطأ
func (cl *ComprehensiveLogger) LogError(source, description string, data map[string]interface{}) {
	log := &SystemLog{
		ID:          generateChatID(),
		Type:        "error",
		Source:      source,
		Description: description,
		Data:        data,
		Timestamp:   time.Now(),
		Severity:    "error",
	}

	cl.recordLog(log)
}

// LogInfo يسجل معلومة
func (cl *ComprehensiveLogger) LogInfo(source, description string, data map[string]interface{}) {
	log := &SystemLog{
		ID:          generateChatID(),
		Type:        "info",
		Source:      source,
		Description: description,
		Data:        data,
		Timestamp:   time.Now(),
		Severity:    "info",
	}

	cl.recordLog(log)
}

// LogWarning يسجل تحذير
func (cl *ComprehensiveLogger) LogWarning(source, description string, data map[string]interface{}) {
	log := &SystemLog{
		ID:          generateChatID(),
		Type:        "warning",
		Source:      source,
		Description: description,
		Data:        data,
		Timestamp:   time.Now(),
		Severity:    "warning",
	}

	cl.recordLog(log)
}

// LogCritical يسجل خطأ حرج
func (cl *ComprehensiveLogger) LogCritical(source, description string, data map[string]interface{}) {
	log := &SystemLog{
		ID:          generateChatID(),
		Type:        "critical",
		Source:      source,
		Description: description,
		Data:        data,
		Timestamp:   time.Now(),
		Severity:    "critical",
	}

	cl.recordLog(log)
}

// ============================================================
// معالجة السجلات
// ============================================================

// recordLog يسجل سجل ويبثه
func (cl *ComprehensiveLogger) recordLog(log *SystemLog) {
	cl.mu.Lock()
	cl.logs[log.ID] = log
	cl.mu.Unlock()

	cl.logger.Info("تم تسجيل سجل مباشرة",
		zap.String("log_id", log.ID),
		zap.String("type", log.Type),
		zap.String("source", log.Source),
		zap.Int("total_logs", len(cl.logs)),
	)

	// [FIX] التأكد من أن السجل تم حفظه بشكل صحيح
	cl.mu.RLock()
	_, exists := cl.logs[log.ID]
	cl.mu.RUnlock()

	if !exists {
		cl.logger.Error("فشل حفظ السجل", zap.String("log_id", log.ID))
		return
	}

	cl.logToEventBus <- log

	cl.mu.Lock()
	cl.metrics.LogsRecorded++
	cl.metrics.LastActivity = time.Now()
	cl.mu.Unlock()
}

// ============================================================
// معالجة الرسائل
// ============================================================

// subscribeToEventBus يرتبط بأحداث Event Bus
func (cl *ComprehensiveLogger) subscribeToEventBus() {
	cl.eventBus.Subscribe("*", cl.handleAllEvents) // الاشتراك في جميع الأحداث
}

// logHandler يعالج السجلات
func (cl *ComprehensiveLogger) logHandler() {
	defer cl.wg.Done()

	for {
		select {
		case <-cl.ctx.Done():
			return
		case log := <-cl.logToEventBus:
			cl.processLog(log)
		}
	}
}

// processLog يعالج سجل
func (cl *ComprehensiveLogger) processLog(log *SystemLog) {
	// تحويل السجل إلى حدث Event Bus
	event := eventbus.Event{
		Type:      "system.log",
		Payload:   log,
		Source:    log.Source,
		SessionID: log.SessionID,
		Timestamp: log.Timestamp,
	}

	// نشر الحدث
	cl.eventBus.Publish(event)

	cl.mu.Lock()
	cl.metrics.LogsBroadcasted++
	cl.mu.Unlock()

	// بث السجل لجميع الوكلاء
	cl.broadcastToAgents(log)
}

// broadcastToAgents يبث السجل لجميع الوكلاء
func (cl *ComprehensiveLogger) broadcastToAgents(log *SystemLog) {
	// إنشاء حدث بث
	broadcastEvent := eventbus.Event{
		Type:      "log.broadcast",
		Payload:   log,
		Source:    "system",
		Timestamp: time.Now(),
	}

	// نشر الحدث لجميع الوكلاء
	cl.eventBus.Publish(broadcastEvent)
}

// eventBusHandler يعالج أحداث Event Bus
func (cl *ComprehensiveLogger) eventBusHandler() {
	defer cl.wg.Done()

	for {
		select {
		case <-cl.ctx.Done():
			return
		case event := <-cl.eventBusToLog:
			cl.processEventBusEvent(event)
		}
	}
}

// processEventBusEvent يعالج حدث Event Bus
func (cl *ComprehensiveLogger) processEventBusEvent(event eventbus.Event) {
	// تسجيل جميع الأحداث التي تمر عبر Event Bus
	log := &SystemLog{
		ID:          generateChatID(),
		Type:        "system_event",
		Source:      event.Source,
		Description: fmt.Sprintf("حدث: %s", event.Type),
		Data: map[string]interface{}{
			"event_type": event.Type,
			"payload":    event.Payload,
		},
		Timestamp: event.Timestamp,
		Severity:  "debug",
	}

	cl.recordLog(log)
}

// handleAllEvents يعالج جميع الأحداث
func (cl *ComprehensiveLogger) handleAllEvents(event eventbus.Event) {
	// تسجيل الحدث تلقائياً
	cl.processEventBusEvent(event)
}

// ============================================================
// استرجاع السجلات
// ============================================================

// GetLogs يحصل على جميع السجلات
func (cl *ComprehensiveLogger) GetLogs() []*SystemLog {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	logs := make([]*SystemLog, 0, len(cl.logs))
	for _, log := range cl.logs {
		logs = append(logs, log)
	}

	return logs
}

// GetLogsByType يحصل على سجلات حسب النوع
func (cl *ComprehensiveLogger) GetLogsByType(logType string) []*SystemLog {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	var logs []*SystemLog
	for _, log := range cl.logs {
		if log.Type == logType {
			logs = append(logs, log)
		}
	}

	return logs
}

// GetLogsBySource يحصل على سجلات حسب المصدر
func (cl *ComprehensiveLogger) GetLogsBySource(source string) []*SystemLog {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	var logs []*SystemLog
	for _, log := range cl.logs {
		if log.Source == source {
			logs = append(logs, log)
		}
	}

	return logs
}

// GetLogsBySession يحصل على سجلات حسب الجلسة
func (cl *ComprehensiveLogger) GetLogsBySession(sessionID string) []*SystemLog {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	var logs []*SystemLog
	for _, log := range cl.logs {
		// [FIX] التحقق من SessionID في البيانات أيضاً
		if log.SessionID == sessionID {
			logs = append(logs, log)
		} else if log.Data != nil {
			if sid, ok := log.Data["session_id"]; ok && sid == sessionID {
				logs = append(logs, log)
			}
		}
	}

	return logs
}

// ============================================================
// المقاييس
// ============================================================

// GetMetrics يحصل على المقاييس
func (cl *ComprehensiveLogger) GetMetrics() *LoggerMetrics {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	return &LoggerMetrics{
		LogsRecorded:    cl.metrics.LogsRecorded,
		LogsBroadcasted: cl.metrics.LogsBroadcasted,
		Errors:          cl.metrics.Errors,
		LastActivity:    cl.metrics.LastActivity,
	}
}

// ============================================================
// تصدير السجلات
// ============================================================

// ExportLogsToJSON يصدر السجلات إلى JSON
func (cl *ComprehensiveLogger) ExportLogsToJSON() ([]byte, error) {
	logs := cl.GetLogs()
	return json.MarshalIndent(logs, "", "  ")
}
