package plugins

import (
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/plugins/core"
	"go.uber.org/zap"
)

// PluginsIntegrator يربط pkg/plugins مع النظام الحالي
type PluginsIntegrator struct {
	manager *core.PluginManager
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

// NewPluginsIntegrator ينشئ تكاملاً جديداً للإضافات
func NewPluginsIntegrator(logger *zap.Logger, eventBus *eventbus.EventBus) *PluginsIntegrator {
	// إنشاء محول لـ EventBus
	adapter := &EventBusAdapter{eb: eventBus}
	
	// إنشاء PluginManager
	manager := core.NewPluginManager(logger, adapter)
	
	return &PluginsIntegrator{
		manager: manager,
		logger:  logger,
	}
}

// GetManager يحصل على PluginManager
func (pi *PluginsIntegrator) GetManager() *core.PluginManager {
	return pi.manager
}

// Start يبدأ تكامل الإضافات
func (pi *PluginsIntegrator) Start() error {
	pi.logger.Info("بدء تكامل الإضافات")
	return nil
}

// Stop يوقف تكامل الإضافات
func (pi *PluginsIntegrator) Stop() error {
	pi.logger.Info("إيقاف تكامل الإضافات")
	return nil
}
