package unified

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

const eventTypePrefix = "session.event."

// SessionEventBusBridge يربط SessionEventBus مع EventBus الرئيسي + الشبكة
// [WHY] أحداث الوكلاء (ذاكرة، مهارات، رسائل) تتدفق إلى:
//       1. EventBus المحلي (للمشتركين المحليين)
//       2. قناة الشبكة (Outbound) لتصل للأجهزة الأخرى
//       والأحداث البعيدة تتدفق بالعكس عبر FeedFromNetwork()
//
// [SAFETY] الأحداث البعيدة لا تُعاد توجيهها للشبكة (metadata.remote = true)
//          مما يمنع الحلقات اللانهائية
type SessionEventBusBridge struct {
	sessionID       string
	sessionBus      *SessionEventBus
	mainBus         *eventbus.EventBus
	networkOutbound chan<- eventbus.Event // اختياري: قناة SessionNetworkBridge.Outbound()
	ctx             context.Context
	logger          *zap.Logger
	stopped         chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
	localEvents     int64
	remoteEvents    int64
	active          bool

	// journalCallback اختياري — يُستدعى عند كل حدث لكتابة سجل الجلسة
	journalCallback func(eventType string, sourceID, sourceType, summary string, details interface{})

	// remoteEventSink اختياري — يُدفع إليه SessionEvent البعيد
	// [WHY] يُوصَل بـ AgentSyncManager.RemoteEventChannel() للذاكرة/المهارات عن بعد
	remoteEventSink chan<- *SessionEvent
}

// NewSessionEventBusBridge ينشئ جسراً بين SessionEventBus و EventBus
// إذا كانت networkOutbound != nil، الأحداث المحلية ترسل مباشرة للشبكة أيضاً
func NewSessionEventBusBridge(ctx context.Context, sessionID string, sessionBus *SessionEventBus, mainBus *eventbus.EventBus, logger *zap.Logger) *SessionEventBusBridge {
	return newBridge(ctx, sessionID, sessionBus, mainBus, nil, nil, nil, logger)
}

// NewSessionEventBusBridgeWithNetwork ينشئ جسراً مع قناة شبكة لإرسال الأحداث للأجهزة الأخرى
func NewSessionEventBusBridgeWithNetwork(ctx context.Context, sessionID string, sessionBus *SessionEventBus, mainBus *eventbus.EventBus, networkOutbound chan<- eventbus.Event, logger *zap.Logger) *SessionEventBusBridge {
	out := networkOutbound
	return newBridge(ctx, sessionID, sessionBus, mainBus, &out, nil, nil, logger)
}

// NewSessionEventBusBridgeFull ينشئ جسراً كامل الخيارات (journal + remote sink)
func NewSessionEventBusBridgeFull(ctx context.Context, sessionID string, sessionBus *SessionEventBus, mainBus *eventbus.EventBus, networkOutbound chan<- eventbus.Event, journalCB func(string, string, string, string, interface{}), remoteSink chan<- *SessionEvent, logger *zap.Logger) *SessionEventBusBridge {
	return newBridge(ctx, sessionID, sessionBus, mainBus, &networkOutbound, journalCB, remoteSink, logger)
}

// SetJournalCallback يضبط دالة كتابة السجل بعد الإنشاء
func (b *SessionEventBusBridge) SetJournalCallback(cb func(eventType string, sourceID, sourceType, summary string, details interface{})) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.journalCallback = cb
}

// SetRemoteEventSink يضبط قناة الأحداث البعيدة بعد الإنشاء
func (b *SessionEventBusBridge) SetRemoteEventSink(sink chan<- *SessionEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.remoteEventSink = sink
}

func newBridge(ctx context.Context, sessionID string, sessionBus *SessionEventBus, mainBus *eventbus.EventBus, networkOutbound *chan<- eventbus.Event, journalCB func(string, string, string, string, interface{}), remoteSink chan<- *SessionEvent, logger *zap.Logger) *SessionEventBusBridge {
	b := &SessionEventBusBridge{
		sessionID:       sessionID,
		sessionBus:      sessionBus,
		mainBus:         mainBus,
		ctx:             ctx,
		logger:          logger,
		stopped:         make(chan struct{}),
		active:          true,
		journalCallback: journalCB,
		remoteEventSink: remoteSink,
	}
	if networkOutbound != nil {
		b.networkOutbound = *networkOutbound
	}

	b.wg.Add(1)
	go b.forwardLocalEvents()

	logger.Info("جسر SessionEventBus ← EventBus + شبكة نشط",
		zap.String("session_id", sessionID),
		zap.Bool("has_network", networkOutbound != nil))
	return b
}

// Close يوقف الجسر
func (b *SessionEventBusBridge) Close() {
	b.mu.Lock()
	if !b.active {
		b.mu.Unlock()
		return
	}
	b.active = false
	b.mu.Unlock()
	close(b.stopped)
	b.wg.Wait()
}

// FeedFromNetwork يُغذّي حدثاً قادماً من الشبكة (عبر EventBus) إلى SessionEventBus
// [WHY] يُستدعى من SessionNetworkBridge عند استقبال حدث بعيد بنمط session.event.*
func (b *SessionEventBusBridge) FeedFromNetwork(evt eventbus.Event) {
	if !strings.HasPrefix(evt.Type, eventTypePrefix) {
		return
	}

	eventType := strings.TrimPrefix(evt.Type, eventTypePrefix)
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		return
	}

	sessionEvent := &SessionEvent{
		ID:          generateID(),
		SessionID:   b.sessionID,
		SourceAgent: extractString(payload, "source_agent"),
		TargetAgent: extractString(payload, "target_agent"),
		EventType:   SessionEventType(eventType),
		Timestamp:   time.Now(),
		Data:        payload["data"],
		Metadata: map[string]interface{}{
			"remote":      true,
			"source_node": evt.Source,
		},
	}

	_ = b.sessionBus.PublishEvent(b.ctx, sessionEvent)

	// دفع الحدث البعيد إلى AgentSyncManager.RemoteEventChannel()
	b.mu.RLock()
	sink := b.remoteEventSink
	b.mu.RUnlock()
	if sink != nil {
		select {
		case sink <- sessionEvent:
		default:
		}
	}

	// تسجيل في سجل الجلسة
	b.mu.RLock()
	cb := b.journalCallback
	b.mu.RUnlock()
	if cb != nil {
		cb("event.logged", sessionEvent.SourceAgent, "agent",
			"حدث بعيد: "+string(sessionEvent.EventType),
			map[string]interface{}{
				"event_type": sessionEvent.EventType,
				"source":     sessionEvent.SourceAgent,
				"target":     sessionEvent.TargetAgent,
			})
	}

	b.mu.Lock()
	b.remoteEvents++
	b.mu.Unlock()
}

// GetStats يرجع إحصائيات الجسر
func (b *SessionEventBusBridge) GetStats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return map[string]interface{}{
		"local_events":  b.localEvents,
		"remote_events": b.remoteEvents,
		"active":        b.active,
	}
}

// forwardLocalEvents يقرأ كل أحداث SessionEventBus ويُعيد توجيهها إلى:
// 1. EventBus المحلي
// 2. قناة الشبكة (إذا كانت موجودة)
func (b *SessionEventBusBridge) forwardLocalEvents() {
	defer b.wg.Done()

	smChan := b.sessionBus.GetSessionManagerChannel()
	for {
		b.mu.RLock()
		active := b.active
		b.mu.RUnlock()
		if !active {
			return
		}

		select {
		case <-b.stopped:
			return
		case sessionEvent, ok := <-smChan:
			if !ok {
				return
			}

			// تجاهل الأحداث البعيدة (منع الحلقات اللانهائية)
			if sessionEvent.Metadata != nil {
				if remote, _ := sessionEvent.Metadata["remote"].(bool); remote {
					continue
				}
			}

			// بناء حدث EventBus
			mainEvent := eventbus.Event{
				Type:      eventTypePrefix + string(sessionEvent.EventType),
				Source:    sessionEvent.SourceAgent,
				SessionID: b.sessionID,
				Payload: map[string]interface{}{
					"session_event_id": sessionEvent.ID,
					"event_type":       string(sessionEvent.EventType),
					"source_agent":     sessionEvent.SourceAgent,
					"target_agent":     sessionEvent.TargetAgent,
					"data":             sessionEvent.Data,
					"metadata":         sessionEvent.Metadata,
					"priority":         string(sessionEvent.Priority),
				},
				Timestamp: sessionEvent.Timestamp,
			}

			// 1. EventBus المحلي — النوع المحدد
			b.mainBus.Publish(mainEvent)

			// 1b. EventBus المحلي — النوع العام (لـ WebSocket UI)
			genericEvent := mainEvent
			genericEvent.Type = "session.agent_event"
			b.mainBus.Publish(genericEvent)

			// 2. الشبكة (إذا كانت القناة موجودة)
			if b.networkOutbound != nil {
				select {
				case b.networkOutbound <- mainEvent:
				default:
					b.logger.Warn("قناة الشبكة ممتلئة، تم تجاهل حدث وكيل")
				}
			}

			// 3. تسجيل في سجل الجلسة
			b.mu.RLock()
			cb := b.journalCallback
			b.mu.RUnlock()
			if cb != nil {
				cb("event.logged", sessionEvent.SourceAgent, "agent",
					"حدث محلي: "+string(sessionEvent.EventType),
					map[string]interface{}{
						"event_type": sessionEvent.EventType,
						"source":     sessionEvent.SourceAgent,
						"target":     sessionEvent.TargetAgent,
					})
			}

			b.mu.Lock()
			b.localEvents++
			b.mu.Unlock()
		}
	}
}

func extractString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

// EventTypeForSessionEvent يرجع اسم حدث EventBus المناظر لنوع حدث جلسة
func EventTypeForSessionEvent(et SessionEventType) string {
	return eventTypePrefix + string(et)
}

// IsSessionEventType يتحقق مما إذا كان نوع حدث يبدأ بـ session.event.
func IsSessionEventType(eventType string) bool {
	return strings.HasPrefix(eventType, eventTypePrefix)
}
