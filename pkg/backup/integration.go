package backup

import (
	"github.com/MortalArena/Musketeers/pkg/backup/core"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

// BackupIntegrator يربط pkg/backup مع النظام الحالي
type BackupIntegrator struct {
	manager *core.BackupManager
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

// NewBackupIntegrator ينشئ تكاملاً جديداً للنسخ الاحتياطي
func NewBackupIntegrator(logger *zap.Logger, eventBus *eventbus.EventBus) *BackupIntegrator {
	// إنشاء محول لـ EventBus
	adapter := &EventBusAdapter{eb: eventBus}
	
	// إنشاء تكوين افتراضي
	config := &core.BackupConfig{
		BackupDir:     "./backups",
		RetentionDays: 30,
		MaxBackups:    10,
		Compression:   true,
		Encryption:    false,
		CheckInterval: 0,
	}
	
	// إنشاء BackupManager بدون storage (يمكن إضافته لاحقاً)
	manager := core.NewBackupManager(logger, nil, adapter, config)
	
	return &BackupIntegrator{
		manager: manager,
		logger:  logger,
	}
}

// GetManager يحصل على BackupManager
func (bi *BackupIntegrator) GetManager() *core.BackupManager {
	return bi.manager
}

// Start يبدأ تكامل النسخ الاحتياطي
func (bi *BackupIntegrator) Start() error {
	bi.logger.Info("بدء تكامل النسخ الاحتياطي")
	return nil
}

// Stop يوقف تكامل النسخ الاحتياطي
func (bi *BackupIntegrator) Stop() error {
	bi.logger.Info("إيقاف تكامل النسخ الاحتياطي")
	return nil
}
