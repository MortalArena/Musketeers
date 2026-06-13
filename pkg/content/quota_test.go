package content

import (
	"testing"
)

func TestQuotaManager_Enforcement(t *testing.T) {
	qm := NewQuotaManager()
	did := "did:mskt:test123"

	// تحديد حد صغير للاختبار: 100 بايت
	qm.SetLimit(did, 100)

	// 1. إضافة 60 بايت (يجب أن تنجح)
	err := qm.CheckAndAdd(did, 60)
	if err != nil {
		t.Fatalf("Unexpected error on first addition: %v", err)
	}

	// 2. إضافة 50 بايت أخرى (يجب أن تفشل لأن المجموع 110 > 100)
	err = qm.CheckAndAdd(did, 50)
	if err == nil {
		t.Errorf("Expected quota exceeded error, but got nil")
	}

	// 3. تحرير 30 بايت
	qm.Release(did, 30)

	// 4. إضافة 30 بايت الآن (يجب أن تنجح لأن الاستخدام أصبح 30 + 30 = 60 <= 100)
	err = qm.CheckAndAdd(did, 30)
	if err != nil {
		t.Fatalf("Unexpected error after release: %v", err)
	}
}
