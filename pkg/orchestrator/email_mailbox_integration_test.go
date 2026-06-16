package orchestrator

import (
	"testing"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestEmailManagerWithMailboxIntegration(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager مع mailbox (store = nil)
	emailManager := NewEmailManager(eventBus, nil, zap.NewNop())

	if emailManager == nil {
		t.Fatal("فشل إنشاء EmailManager")
	}

	// التحقق من أن mailbox لم يتم إنشاؤه عندما store هو nil
	if emailManager.mailbox != nil {
		t.Fatal("يجب أن لا يتم إنشاء mailbox عندما store هو nil")
	}

	t.Log("تم إنشاء EmailManager بدون mailbox بنجاح")
}

func TestEmailManagerMailboxFieldExists(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, nil, zap.NewNop())

	// التحقق من وجود حقل mailbox
	if emailManager.mailbox == nil {
		t.Log("mailbox field موجود ولكن nil (صحيح عندما store هو nil)")
	} else {
		t.Error("mailbox يجب أن يكون nil عندما store هو nil")
	}

	t.Log("تم التحقق من وجود حقل mailbox")
}

func TestEmailManagerStoreFieldExists(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء EmailManager
	emailManager := NewEmailManager(eventBus, nil, zap.NewNop())

	// التحقق من وجود حقل store
	if emailManager.store == nil {
		t.Log("store field موجود ولكن nil (صحيح)")
	} else {
		t.Error("store يجب أن يكون nil")
	}

	t.Log("تم التحقق من وجود حقل store")
}
