package content

import (
	"fmt"
	"sync"
)

const DefaultQuotaBytes = 10 * 1024 * 1024 * 1024 // 10 GB حد افتراضي

// QuotaManager يدير حدود التخزين لكل DID بشكل آمن ومتزامن
type QuotaManager struct {
	mu     sync.RWMutex
	limits map[string]int64 // DID -> Max Bytes
	usage  map[string]int64 // DID -> Current Bytes
}

// NewQuotaManager ينشئ مدير حصص جديد
func NewQuotaManager() *QuotaManager {
	return &QuotaManager{
		limits: make(map[string]int64),
		usage:  make(map[string]int64),
	}
}

// SetLimit يحدد الحد الأقصى للتخزين لنطاق معين
func (q *QuotaManager) SetLimit(did string, limitBytes int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.limits[did] = limitBytes
}

// CheckAndAdd يتحقق من المساحة المتاحة ويضيف الاستخدام إذا كان مسموحاً (Atomic Operation)
func (q *QuotaManager) CheckAndAdd(did string, sizeBytes int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	limit, exists := q.limits[did]
	if !exists {
		limit = DefaultQuotaBytes
		q.limits[did] = limit
	}

	currentUsage := q.usage[did]
	if currentUsage+sizeBytes > limit {
		return fmt.Errorf("quota exceeded for DID %s: limit %d, usage %d, requested %d", did, limit, currentUsage, sizeBytes)
	}

	q.usage[did] += sizeBytes
	return nil
}

// Release يحرر المساحة عند حذف ملف أو عقدة
func (q *QuotaManager) Release(did string, sizeBytes int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.usage[did] >= sizeBytes {
		q.usage[did] -= sizeBytes
	} else {
		q.usage[did] = 0 // منع الأرقام السلبية
	}
}
