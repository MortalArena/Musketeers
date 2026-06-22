package analytics

import (
	"github.com/MortalArena/Musketeers/pkg/analytics/core"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// AnalyticsIntegrator يربط pkg/analytics مع النظام الحالي
type AnalyticsIntegrator struct {
	manager *core.AnalyticsManager
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

// NewAnalyticsIntegrator ينشئ تكاملاً جديداً للتحليلات
func NewAnalyticsIntegrator(logger *zap.Logger, eventBus *eventbus.EventBus) *AnalyticsIntegrator {
	// إنشاء محول لـ EventBus
	adapter := &EventBusAdapter{eb: eventBus}
	
	// إنشاء AnalyticsManager بدون storage (يمكن إضافته لاحقاً)
	manager := core.NewAnalyticsManager(logger, nil, adapter)
	
	return &AnalyticsIntegrator{
		manager: manager,
		logger:  logger,
	}
}

// GetManager يحصل على AnalyticsManager
func (ai *AnalyticsIntegrator) GetManager() *core.AnalyticsManager {
	return ai.manager
}

// Start يبدأ تكامل التحليلات
func (ai *AnalyticsIntegrator) Start() error {
	ai.logger.Info("بدء تكامل التحليلات")
	return nil
}

// Stop يوقف تكامل التحليلات
func (ai *AnalyticsIntegrator) Stop() error {
	ai.logger.Info("إيقاف تكامل التحليلات")
	return nil
}
