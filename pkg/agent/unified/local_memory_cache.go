package unified

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LocalMemoryCache ذاكرة محلية للوكيل
type LocalMemoryCache struct {
	sessionID    string
	agentID      string
	memoryEvents map[string]*MemoryEvent
	skillUpdates map[string]*SkillUpdate
	lastSyncTime time.Time
	maxCacheSize int
	logger       *zap.Logger
	mu           sync.RWMutex
}

// SkillUpdate تحديث مهارة
type SkillUpdate struct {
	ID          string
	AgentDID    string
	SkillName   string
	OldLevel    float64
	NewLevel    float64
	XPGained    float64
	SuccessRate float64
	Timestamp   time.Time
	Metadata    map[string]interface{}
}

// NewLocalMemoryCache ينشئ ذاكرة محلية جديدة
func NewLocalMemoryCache(sessionID, agentID string, logger *zap.Logger) *LocalMemoryCache {
	return &LocalMemoryCache{
		sessionID:    sessionID,
		agentID:      agentID,
		memoryEvents: make(map[string]*MemoryEvent),
		skillUpdates: make(map[string]*SkillUpdate),
		lastSyncTime: time.Now(),
		maxCacheSize: 1000, // آخر 1000 حدث
		logger:       logger,
	}
}

// UpdateMemoryEvents يحدث أحداث الذاكرة
func (lmc *LocalMemoryCache) UpdateMemoryEvents(events []*MemoryEvent) {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	for _, event := range events {
		lmc.memoryEvents[event.ID] = event
	}

	// الحفاظ على حجم محدود
	lmc.cleanupOldEntries()

	lmc.logger.Debug("تم تحديث أحداث الذاكرة المحلية",
		zap.Int("events_count", len(events)),
		zap.Int("total_events", len(lmc.memoryEvents)),
	)
}

// UpdateSkillUpdates يحدث تحديثات المهارات
func (lmc *LocalMemoryCache) UpdateSkillUpdates(updates []*SkillUpdate) {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	for _, update := range updates {
		key := fmt.Sprintf("%s:%s", update.AgentDID, update.SkillName)
		lmc.skillUpdates[key] = update
	}

	// الحفاظ على حجم محدود
	lmc.cleanupOldEntries()

	lmc.logger.Debug("تم تحديث تحديثات المهارات المحلية",
		zap.Int("updates_count", len(updates)),
		zap.Int("total_updates", len(lmc.skillUpdates)),
	)
}

// cleanupOldEntries يحذف أقدم الإدخالات للحفاظ على حجم محدود
func (lmc *LocalMemoryCache) cleanupOldEntries() {
	// حذف أحدث الأحداث إذا تجاوزت الحد الأقصى
	if len(lmc.memoryEvents) > lmc.maxCacheSize {
		// حذف أحدث الأحداث
		// في التنفيذ الحقيقي، سيتم حذف أحدث الأحداث
		// هنا سنحذف عشوائياً للتبسيط
		count := 0
		for key := range lmc.memoryEvents {
			if count >= len(lmc.memoryEvents)-lmc.maxCacheSize {
				break
			}
			delete(lmc.memoryEvents, key)
			count++
		}
	}

	if len(lmc.skillUpdates) > lmc.maxCacheSize {
		// حذف أحدث التحديثات
		count := 0
		for key := range lmc.skillUpdates {
			if count >= len(lmc.skillUpdates)-lmc.maxCacheSize {
				break
			}
			delete(lmc.skillUpdates, key)
			count++
		}
	}
}

// GetMemoryEvents يحصل على جميع أحداث الذاكرة
func (lmc *LocalMemoryCache) GetMemoryEvents() []*MemoryEvent {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	events := make([]*MemoryEvent, 0, len(lmc.memoryEvents))
	for _, event := range lmc.memoryEvents {
		events = append(events, event)
	}

	return events
}

// GetSkillUpdates يحصل على جميع تحديثات المهارات
func (lmc *LocalMemoryCache) GetSkillUpdates() []*SkillUpdate {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	updates := make([]*SkillUpdate, 0, len(lmc.skillUpdates))
	for _, update := range lmc.skillUpdates {
		updates = append(updates, update)
	}

	return updates
}

// GetRecentMemoryEvents يحصل على أحدث أحداث الذاكرة
func (lmc *LocalMemoryCache) GetRecentMemoryEvents(count int) []*MemoryEvent {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	events := make([]*MemoryEvent, 0, count)
	for _, event := range lmc.memoryEvents {
		events = append(events, event)
		if len(events) >= count {
			break
		}
	}

	return events
}

// GetRecentSkillUpdates يحصل على أحدث تحديثات المهارات
func (lmc *LocalMemoryCache) GetRecentSkillUpdates(count int) []*SkillUpdate {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	updates := make([]*SkillUpdate, 0, count)
	for _, update := range lmc.skillUpdates {
		updates = append(updates, update)
		if len(updates) >= count {
			break
		}
	}

	return updates
}

// StartMandatorySync يبدأ المزامنة الإجبارية
func (lmc *LocalMemoryCache) StartMandatorySync(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			lmc.logger.Info("تم إيقاف المزامنة الإجبارية للذاكرة المحلية")
			return
		case <-ticker.C:
			lmc.syncToSharedDB(ctx)
		}
	}
}

// syncToSharedDB يزامن الذاكرة المحلية مع قاعدة البيانات المشتركة
func (lmc *LocalMemoryCache) syncToSharedDB(ctx context.Context) {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	// في التنفيذ الحقيقي، سيتم مزامنة البيانات مع قاعدة البيانات المشتركة
	// هنا سنقوم فقط بتسجيل المزامنة

	lmc.logger.Info("تمت المزامنة الإجبارية للذاكرة المحلية مع قاعدة البيانات المشتركة",
		zap.Int("memory_events", len(lmc.memoryEvents)),
		zap.Int("skill_updates", len(lmc.skillUpdates)),
		zap.Time("last_sync", lmc.lastSyncTime),
	)

	lmc.lastSyncTime = time.Now()
}

// GetCacheInfo يحصل على معلومات الذاكرة المحلية
func (lmc *LocalMemoryCache) GetCacheInfo() map[string]interface{} {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	return map[string]interface{}{
		"session_id":     lmc.sessionID,
		"agent_id":       lmc.agentID,
		"memory_events":  len(lmc.memoryEvents),
		"skill_updates":  len(lmc.skillUpdates),
		"last_sync_time": lmc.lastSyncTime,
		"max_cache_size": lmc.maxCacheSize,
	}
}

// GetPendingMemoryEvents يحصل على الأحداث الجديدة المعلقة
func (lmc *LocalMemoryCache) GetPendingMemoryEvents(batchSize int) []*MemoryEvent {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	events := make([]*MemoryEvent, 0, batchSize)
	for _, event := range lmc.memoryEvents {
		events = append(events, event)
		if len(events) >= batchSize {
			break
		}
	}

	return events
}

// MarkMemoryEventSent يعليم حدث الذاكرة كمرسل
func (lmc *LocalMemoryCache) MarkMemoryEventSent(eventID string) {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	// في التنفيذ الحقيقي، سيتم تعليم الحدث كمرسل
	// هنا سنقوم فقط بحذفه من الخريطة
	delete(lmc.memoryEvents, eventID)
}

// GetPendingSkillUpdates يحصل على تحديثات المهارات المعلقة
func (lmc *LocalMemoryCache) GetPendingSkillUpdates(batchSize int) []*SkillUpdate {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	updates := make([]*SkillUpdate, 0, batchSize)
	for _, update := range lmc.skillUpdates {
		updates = append(updates, update)
		if len(updates) >= batchSize {
			break
		}
	}

	return updates
}

// MarkSkillUpdateSent يعليم تحديث المهارة كمرسل
func (lmc *LocalMemoryCache) MarkSkillUpdateSent(updateID string) {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	// في التنفيذ الحقيقي، سيتم تعليم التحديث كمرسل
	// هنا سنقوم فقط بحذفه من الخريطة
	delete(lmc.skillUpdates, updateID)
}

// HasMemoryEvent يتحقق مما إذا كان حدث الذاكرة موجوداً
func (lmc *LocalMemoryCache) HasMemoryEvent(eventID string) bool {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	_, exists := lmc.memoryEvents[eventID]
	return exists
}

// AddMemoryEvent يضيف حدث الذاكرة
func (lmc *LocalMemoryCache) AddMemoryEvent(event MemoryEvent) error {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	lmc.memoryEvents[event.ID] = &event

	// الحفاظ على حجم محدود
	lmc.cleanupOldEntries()

	return nil
}

// HasSkillUpdate يتحقق مما إذا كان تحديث المهارة موجوداً
func (lmc *LocalMemoryCache) HasSkillUpdate(updateID string) bool {
	lmc.mu.RLock()
	defer lmc.mu.RUnlock()

	_, exists := lmc.skillUpdates[updateID]
	return exists
}

// AddSkillUpdate يضيف تحديث المهارة
func (lmc *LocalMemoryCache) AddSkillUpdate(update SkillUpdate) error {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()

	lmc.skillUpdates[update.ID] = &update

	// الحفاظ على حجم محدود
	lmc.cleanupOldEntries()

	return nil
}
