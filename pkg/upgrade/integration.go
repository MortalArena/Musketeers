package upgrade

import (
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/MortalArena/Musketeers/pkg/upgrade/core"
	"go.uber.org/zap"
)

// UpgradeIntegrator يربط pkg/upgrade مع النظام الحالي
type UpgradeIntegrator struct {
	manager *core.UpgradeManager
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

// NewUpgradeIntegrator ينشئ تكاملاً جديداً للترقية
func NewUpgradeIntegrator(logger *zap.Logger, eventBus *eventbus.EventBus) *UpgradeIntegrator {
	// إنشاء محول لـ EventBus
	adapter := &EventBusAdapter{eb: eventBus}
	
	// إنشاء تكوين افتراضي
	config := &core.UpgradeConfig{
		AutoCheck:     true,
		CheckInterval: 24 * time.Hour,
		AutoDownload:  false,
		AutoInstall:   false,
		BackupBefore:  true,
		Channel:       "stable",
	}
	
	// إنشاء UpgradeManager بدون storage (يمكن إضافته لاحقاً)
	manager := core.NewUpgradeManager(logger, nil, adapter, config)
	
	return &UpgradeIntegrator{
		manager: manager,
		logger:  logger,
	}
}

// GetManager يحصل على UpgradeManager
func (ui *UpgradeIntegrator) GetManager() *core.UpgradeManager {
	return ui.manager
}

// Start يبدأ تكامل الترقية
func (ui *UpgradeIntegrator) Start() error {
	ui.logger.Info("بدء تكامل الترقية")
	return nil
}

// Stop يوقف تكامل الترقية
func (ui *UpgradeIntegrator) Stop() error {
	ui.logger.Info("إيقاف تكامل الترقية")
	return nil
}
