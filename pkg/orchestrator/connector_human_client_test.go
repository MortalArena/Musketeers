package orchestrator

import (
	"testing"

	agent "github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"go.uber.org/zap"
)

func TestConnectorRegisterHumanClient(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// تسجيل عميل بشري
	err := connector.RegisterHumanClient("user-123", "Test User", true)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	t.Log("تم تسجيل عميل بشري بنجاح")
}

func TestConnectorUpdateHumanClientStatus(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// تسجيل عميل بشري
	err := connector.RegisterHumanClient("user-123", "Test User", true)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	// تحديث حالة العميل البشري
	err = connector.UpdateHumanClientStatus("offline")
	if err != nil {
		t.Fatalf("فشل تحديث حالة العميل البشري: %v", err)
	}

	t.Log("تم تحديث حالة العميل البشري بنجاح")
}

func TestConnectorGetHumanClientStatus(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// تسجيل عميل بشري
	err := connector.RegisterHumanClient("user-123", "Test User", true)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	// الحصول على حالة العميل البشري
	status, err := connector.GetHumanClientStatus()
	if err != nil {
		t.Fatalf("فشل الحصول على حالة العميل البشري: %v", err)
	}

	if status.UserID != "user-123" {
		t.Errorf("User ID غير صحيح: got %s, want user-123", status.UserID)
	}

	if status.Name != "Test User" {
		t.Errorf("Name غير صحيح: got %s, want Test User", status.Name)
	}

	if status.Status != "online" {
		t.Errorf("Status غير صحيح: got %s, want online", status.Status)
	}

	if !status.AllowOnline {
		t.Error("AllowOnline يجب أن يكون true")
	}

	t.Log("تم الحصول على حالة العميل البشري بنجاح")
}

func TestConnectorSetHumanClientOnlinePreference(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// تسجيل عميل بشري مع allowOnline = true
	err := connector.RegisterHumanClient("user-123", "Test User", true)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	// تغيير تفضيل العميل البشري إلى false
	err = connector.SetHumanClientOnlinePreference(false)
	if err != nil {
		t.Fatalf("فشل تغيير تفضيل العميل البشري: %v", err)
	}

	// الحصول على حالة العميل البشري
	status, err := connector.GetHumanClientStatus()
	if err != nil {
		t.Fatalf("فشل الحصول على حالة العميل البشري: %v", err)
	}

	if status.AllowOnline {
		t.Error("AllowOnline يجب أن يكون false")
	}

	if status.Status != "offline" {
		t.Errorf("Status يجب أن يكون offline عندما AllowOnline هو false, got %s", status.Status)
	}

	t.Log("تم تغيير تفضيل العميل البشري بنجاح")
}

func TestConnectorHumanClientOnlinePreferenceRespected(t *testing.T) {
	// إنشاء EventBus
	eventBus := eventbus.NewEventBus()

	// إنشاء AgentRegistry
	agentRegistry := agent.NewAgentRegistry()
	agentRegistry.SetLogger(zap.NewNop())

	// إنشاء Connector
	connector := NewConnector(eventBus, nil, agentRegistry, zap.NewNop())

	// بدء Connector
	if err := connector.Start(); err != nil {
		t.Fatalf("فشل بدء Connector: %v", err)
	}
	defer connector.Stop()

	// تسجيل عميل بشري مع allowOnline = false
	err := connector.RegisterHumanClient("user-123", "Test User", false)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	// محاولة تحديث حالة العميل البشري إلى online (يجب أن يفشل)
	err = connector.UpdateHumanClientStatus("online")
	if err == nil {
		t.Error("يجب أن يفشل تحديث الحالة إلى online عندما AllowOnline هو false")
	}

	t.Log("تم احترام تفضيل العميل البشري بنجاح")
}
