package orchestrator

import (
	"testing"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestSessionEventBroadcasterCreation(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// إنشاء SessionEventBroadcaster
	broadcaster := NewSessionEventBroadcaster(eventBus, a2aManager, zap.NewNop())

	if broadcaster == nil {
		t.Fatal("فشل إنشاء SessionEventBroadcaster")
	}

	t.Log("تم إنشاء SessionEventBroadcaster بنجاح")
}

func TestSessionEventBroadcasterStartStop(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// إنشاء SessionEventBroadcaster
	broadcaster := NewSessionEventBroadcaster(eventBus, a2aManager, zap.NewNop())

	// بدء SessionEventBroadcaster
	if err := broadcaster.Start(); err != nil {
		t.Fatalf("فشل بدء SessionEventBroadcaster: %v", err)
	}

	// إيقاف SessionEventBroadcaster
	if err := broadcaster.Stop(); err != nil {
		t.Fatalf("فشل إيقاف SessionEventBroadcaster: %v", err)
	}

	t.Log("تم بدء وإيقاف SessionEventBroadcaster بنجاح")
}

func TestSessionEventBroadcasterBroadcastEvent(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// إنشاء SessionEventBroadcaster
	broadcaster := NewSessionEventBroadcaster(eventBus, a2aManager, zap.NewNop())

	// بدء SessionEventBroadcaster
	if err := broadcaster.Start(); err != nil {
		t.Fatalf("فشل بدء SessionEventBroadcaster: %v", err)
	}
	defer broadcaster.Stop()

	// بث حدث
	event := &SessionEvent{
		ID:          generateChatID(),
		SessionID:   session.ID,
		Type:        "test_event",
		AgentID:     "planner",
		Description: "Test Event",
		Data:        map[string]interface{}{},
		Timestamp:   time.Now(),
		Priority:    "normal",
	}

	if err := broadcaster.BroadcastEvent(event); err != nil {
		t.Fatalf("فشل بث الحدث: %v", err)
	}

	t.Log("تم بث الحدث بنجاح")
}

func TestSessionEventBroadcasterBroadcastTaskAssigned(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// إنشاء SessionEventBroadcaster
	broadcaster := NewSessionEventBroadcaster(eventBus, a2aManager, zap.NewNop())

	// بدء SessionEventBroadcaster
	if err := broadcaster.Start(); err != nil {
		t.Fatalf("فشل بدء SessionEventBroadcaster: %v", err)
	}
	defer broadcaster.Stop()

	// بث حدث توزيع مهمة
	if err := broadcaster.BroadcastTaskAssigned(session.ID, "coder", "Test Task", map[string]interface{}{
		"details": "Task details",
	}); err != nil {
		t.Fatalf("فشل بث حدث توزيع مهمة: %v", err)
	}

	t.Log("تم بث حدث توزيع مهمة بنجاح")
}

func TestSessionEventBroadcasterBroadcastTaskCompleted(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// إنشاء SessionEventBroadcaster
	broadcaster := NewSessionEventBroadcaster(eventBus, a2aManager, zap.NewNop())

	// بدء SessionEventBroadcaster
	if err := broadcaster.Start(); err != nil {
		t.Fatalf("فشل بدء SessionEventBroadcaster: %v", err)
	}
	defer broadcaster.Stop()

	// إنشاء artifact
	artifact := &A2AArtifact{
		ID:        generateChatID(),
		Type:      "code",
		Name:      "Test Artifact",
		Content:   "Test Content",
		CreatedBy: "coder",
		CreatedAt: time.Now(),
	}

	// بث حدث إكمال مهمة
	if err := broadcaster.BroadcastTaskCompleted(session.ID, "coder", []*A2AArtifact{artifact}); err != nil {
		t.Fatalf("فشل بث حدث إكمال مهمة: %v", err)
	}

	t.Log("تم بث حدث إكمال مهمة بنجاح")
}

func TestSessionEventBroadcasterBroadcastProgressUpdate(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// إنشاء SessionEventBroadcaster
	broadcaster := NewSessionEventBroadcaster(eventBus, a2aManager, zap.NewNop())

	// بدء SessionEventBroadcaster
	if err := broadcaster.Start(); err != nil {
		t.Fatalf("فشل بدء SessionEventBroadcaster: %v", err)
	}
	defer broadcaster.Stop()

	// بث تحديث تقدم
	if err := broadcaster.BroadcastProgressUpdate(session.ID, "coder", 50, "Halfway done"); err != nil {
		t.Fatalf("فشل بث تحديث تقدم: %v", err)
	}

	t.Log("تم بث تحديث تقدم بنجاح")
}

func TestSessionEventBroadcasterBroadcastError(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// بدء A2AManager
	if err := a2aManager.Start(); err != nil {
		t.Fatalf("فشل بدء A2AManager: %v", err)
	}
	defer a2aManager.Stop()

	// إنشاء جلسة
	session, err := a2aManager.CreateSession("task-123", "Test Goal", []string{"planner", "coder"})
	if err != nil {
		t.Fatalf("فشل إنشاء الجلسة: %v", err)
	}

	// إنشاء SessionEventBroadcaster
	broadcaster := NewSessionEventBroadcaster(eventBus, a2aManager, zap.NewNop())

	// بدء SessionEventBroadcaster
	if err := broadcaster.Start(); err != nil {
		t.Fatalf("فشل بدء SessionEventBroadcaster: %v", err)
	}
	defer broadcaster.Stop()

	// بث خطأ
	if err := broadcaster.BroadcastError(session.ID, "coder", "Test Error", map[string]interface{}{
		"context": "Error context",
	}); err != nil {
		t.Fatalf("فشل بث خطأ: %v", err)
	}

	t.Log("تم بث خطأ بنجاح")
}

func TestSessionEventBroadcasterGetMetrics(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء A2AManager
	a2aManager := NewA2AManager(eventBus, zap.NewNop())

	// إنشاء SessionEventBroadcaster
	broadcaster := NewSessionEventBroadcaster(eventBus, a2aManager, zap.NewNop())

	// بدء SessionEventBroadcaster
	if err := broadcaster.Start(); err != nil {
		t.Fatalf("فشل بدء SessionEventBroadcaster: %v", err)
	}
	defer broadcaster.Stop()

	// الحصول على المقاييس
	metrics := broadcaster.GetMetrics()

	if metrics == nil {
		t.Error("يجب أن تكون هناك مقاييس")
	}

	t.Logf("المقاييس: %+v", metrics)
}
