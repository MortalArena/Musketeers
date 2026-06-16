package orchestrator

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestEmailManagerCreation(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	if emailManager == nil {
		t.Fatal("فشل إنشاء EmailManager")
	}

	t.Log("تم إنشاء EmailManager بنجاح")
}

func TestEmailManagerStartStop(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}

	// إيقاف EmailManager
	if err := emailManager.Stop(); err != nil {
		t.Fatalf("فشل إيقاف EmailManager: %v", err)
	}

	t.Log("تم بدء وإيقاف EmailManager بنجاح")
}

func TestEmailSendEmail(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}
	defer emailManager.Stop()

	// إنشاء إيميل
	email := &Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
		Priority: "normal",
	}

	// إرسال الإيميل
	if err := emailManager.SendEmail(email); err != nil {
		t.Fatalf("فشل إرسال الإيميل: %v", err)
	}

	t.Log("تم إرسال الإيميل بنجاح")
}

func TestEmailReceiveEmail(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}
	defer emailManager.Stop()

	// إنشاء إيميل
	email := &Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
		Priority: "normal",
	}

	// استقبال الإيميل
	if err := emailManager.ReceiveEmail(email); err != nil {
		t.Fatalf("فشل استقبال الإيميل: %v", err)
	}

	t.Log("تم استقبال الإيميل بنجاح")
}

func TestEmailReadEmail(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}
	defer emailManager.Stop()

	// إنشاء إيميل
	email := &Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
		Priority: "normal",
	}

	// استقبال الإيميل
	if err := emailManager.ReceiveEmail(email); err != nil {
		t.Fatalf("فشل استقبال الإيميل: %v", err)
	}

	// قراءة الإيميل
	if err := emailManager.ReadEmail(email.ID); err != nil {
		t.Fatalf("فشل قراءة الإيميل: %v", err)
	}

	t.Log("تم قراءة الإيميل بنجاح")
}

func TestEmailDeleteEmail(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}
	defer emailManager.Stop()

	// إنشاء إيميل
	email := &Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
		Priority: "normal",
	}

	// استقبال الإيميل
	if err := emailManager.ReceiveEmail(email); err != nil {
		t.Fatalf("فشل استقبال الإيميل: %v", err)
	}

	// حذف الإيميل
	if err := emailManager.DeleteEmail(email.ID); err != nil {
		t.Fatalf("فشل حذف الإيميل: %v", err)
	}

	t.Log("تم حذف الإيميل بنجاح")
}

func TestEmailMoveEmail(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}
	defer emailManager.Stop()

	// إنشاء إيميل
	email := &Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
		Priority: "normal",
	}

	// استقبال الإيميل
	if err := emailManager.ReceiveEmail(email); err != nil {
		t.Fatalf("فشل استقبال الإيميل: %v", err)
	}

	// نقل الإيميل إلى مجلد آخر
	if err := emailManager.MoveEmail(email.ID, "starred"); err != nil {
		t.Fatalf("فشل نقل الإيميل: %v", err)
	}

	t.Log("تم نقل الإيميل بنجاح")
}

func TestEmailCreateFolder(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}
	defer emailManager.Stop()

	// إنشاء مجلد جديد
	folder, err := emailManager.CreateFolder("Custom Folder", "custom")
	if err != nil {
		t.Fatalf("فشل إنشاء المجلد: %v", err)
	}

	if folder.Name != "Custom Folder" {
		t.Errorf("اسم المجلد غير صحيح: got %s, want Custom Folder", folder.Name)
	}

	t.Log("تم إنشاء المجلد بنجاح")
}

func TestEmailCreateMailingList(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}
	defer emailManager.Stop()

	// إنشاء قائمة بريدية
	list, err := emailManager.CreateMailingList("Test List", []string{"user1@example.com", "user2@example.com"}, "Test Description")
	if err != nil {
		t.Fatalf("فشل إنشاء القائمة البريدية: %v", err)
	}

	if list.Name != "Test List" {
		t.Errorf("اسم القائمة البريدية غير صحيح: got %s, want Test List", list.Name)
	}

	t.Log("تم إنشاء القائمة البريدية بنجاح")
}

func TestEmailSendToMailingList(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}
	defer emailManager.Stop()

	// إنشاء قائمة بريدية
	list, err := emailManager.CreateMailingList("Test List", []string{"user1@example.com", "user2@example.com"}, "Test Description")
	if err != nil {
		t.Fatalf("فشل إنشاء القائمة البريدية: %v", err)
	}

	// إنشاء إيميل
	email := &Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
		Priority: "normal",
	}

	// إرسال الإيميل إلى القائمة البريدية
	if err := emailManager.SendToMailingList(list.ID, email); err != nil {
		t.Fatalf("فشل إرسال الإيميل إلى القائمة البريدية: %v", err)
	}

	t.Log("تم إرسال الإيميل إلى القائمة البريدية بنجاح")
}

func TestEmailGetMetrics(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, zap.NewNop())

	// بدء EmailManager
	if err := emailManager.Start(); err != nil {
		t.Fatalf("فشل بدء EmailManager: %v", err)
	}
	defer emailManager.Stop()

	// الحصول على المقاييس
	metrics := emailManager.GetMetrics()

	if metrics == nil {
		t.Error("يجب أن تكون هناك مقاييس")
	}

	if metrics.FoldersCount == 0 {
		t.Error("يجب أن يكون هناك مجلدات")
	}

	t.Logf("المقاييس: %+v", metrics)
}
