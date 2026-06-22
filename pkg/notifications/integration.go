package notifications

import (
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/notifications/core"
	"go.uber.org/zap"
)

// NotificationsIntegrator يربط pkg/notifications مع النظام الحالي
type NotificationsIntegrator struct {
	manager *core.NotificationManager
	logger  *zap.Logger
}

// EventBusAdapter يحول pkg/eventbus إلى واجهة core.EventBus
type EventBusAdapter struct {
	eb *eventbus.EventBus
}

// Publish ينشر حدثاً
func (a *EventBusAdapter) Publish(event string, data interface{}) error {
	a.eb.Publish(eventbus.Event{
		Type:    event,
		Payload: data,
	})
	return nil
}

// Subscribe يسجل معالجاً لحدث معين
func (a *EventBusAdapter) Subscribe(event string, handler func(data interface{})) error {
	a.eb.Subscribe(event, func(e eventbus.Event) {
		handler(e.Payload)
	})
	return nil
}

// MockNotificationSender محاكي بسيط لـ NotificationSender
type MockNotificationSender struct{}

// SendEmail يرسل إيميل
func (m *MockNotificationSender) SendEmail(to, subject, body string) error {
	// محاكاة بسيطة - في التنفيذ الحقيقي يجب إرسال الإيميل فعلياً
	return nil
}

// SendSMS يرسل SMS
func (m *MockNotificationSender) SendSMS(to, message string) error {
	// محاكاة بسيطة - في التنفيذ الحقيقي يجب إرسال SMS فعلياً
	return nil
}

// SendPush يرسل إشعار push
func (m *MockNotificationSender) SendPush(to, title, body string) error {
	// محاكاة بسيطة - في التنفيذ الحقيقي يجب إرسال push فعلياً
	return nil
}

// SendWebhook يرسل webhook
func (m *MockNotificationSender) SendWebhook(url string, data interface{}) error {
	// محاكاة بسيطة - في التنفيذ الحقيقي يجب إرسال webhook فعلياً
	return nil
}

// NewNotificationsIntegrator ينشئ تكاملاً جديداً للإشعارات
func NewNotificationsIntegrator(logger *zap.Logger, eventBus *eventbus.EventBus) *NotificationsIntegrator {
	// إنشاء محول لـ EventBus
	adapter := &EventBusAdapter{eb: eventBus}
	
	// إنشاء محاكي لـ NotificationSender
	sender := &MockNotificationSender{}
	
	// إنشاء NotificationManager
	manager := core.NewNotificationManager(logger, sender, adapter)
	
	return &NotificationsIntegrator{
		manager: manager,
		logger:  logger,
	}
}

// GetManager يحصل على NotificationManager
func (ni *NotificationsIntegrator) GetManager() *core.NotificationManager {
	return ni.manager
}

// Start يبدأ تكامل الإشعارات
func (ni *NotificationsIntegrator) Start() error {
	ni.logger.Info("بدء تكامل الإشعارات")
	return nil
}

// Stop يوقف تكامل الإشعارات
func (ni *NotificationsIntegrator) Stop() error {
	ni.logger.Info("إيقاف تكامل الإشعارات")
	return nil
}
