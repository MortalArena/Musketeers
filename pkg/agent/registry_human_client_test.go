package agent

import (
	"testing"

	"go.uber.org/zap"
)

func TestRegisterHumanClient(t *testing.T) {
	// إنشاء AgentRegistry
	registry := NewAgentRegistry()
	registry.SetLogger(zap.NewNop())

	// تسجيل عميل بشري جديد
	err := registry.RegisterHumanClient("user-123", "Test User", true)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	t.Log("تم تسجيل عميل بشري بنجاح")
}

func TestUpdateHumanClientStatus(t *testing.T) {
	// إنشاء AgentRegistry
	registry := NewAgentRegistry()
	registry.SetLogger(zap.NewNop())

	// تسجيل عميل بشري
	err := registry.RegisterHumanClient("user-123", "Test User", true)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	// تحديث حالة العميل البشري
	err = registry.UpdateHumanClientStatus("offline")
	if err != nil {
		t.Fatalf("فشل تحديث حالة العميل البشري: %v", err)
	}

	t.Log("تم تحديث حالة العميل البشري بنجاح")
}

func TestGetHumanClientStatus(t *testing.T) {
	// إنشاء AgentRegistry
	registry := NewAgentRegistry()
	registry.SetLogger(zap.NewNop())

	// تسجيل عميل بشري
	err := registry.RegisterHumanClient("user-123", "Test User", true)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	// الحصول على حالة العميل البشري
	status, err := registry.GetHumanClientStatus()
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

func TestSetHumanClientOnlinePreference(t *testing.T) {
	// إنشاء AgentRegistry
	registry := NewAgentRegistry()
	registry.SetLogger(zap.NewNop())

	// تسجيل عميل بشري مع allowOnline = true
	err := registry.RegisterHumanClient("user-123", "Test User", true)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	// تغيير تفضيل العميل البشري إلى false
	err = registry.SetHumanClientOnlinePreference(false)
	if err != nil {
		t.Fatalf("فشل تغيير تفضيل العميل البشري: %v", err)
	}

	// الحصول على حالة العميل البشري
	status, err := registry.GetHumanClientStatus()
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

func TestHumanClientOnlinePreferenceRespected(t *testing.T) {
	// إنشاء AgentRegistry
	registry := NewAgentRegistry()
	registry.SetLogger(zap.NewNop())

	// تسجيل عميل بشري مع allowOnline = false
	err := registry.RegisterHumanClient("user-123", "Test User", false)
	if err != nil {
		t.Fatalf("فشل تسجيل عميل بشري: %v", err)
	}

	// محاولة تحديث حالة العميل البشري إلى online (يجب أن يفشل)
	err = registry.UpdateHumanClientStatus("online")
	if err == nil {
		t.Error("يجب أن يفشل تحديث الحالة إلى online عندما AllowOnline هو false")
	}

	t.Log("تم احترام تفضيل العميل البشري بنجاح")
}

func TestHumanClientNotRegistered(t *testing.T) {
	// إنشاء AgentRegistry
	registry := NewAgentRegistry()
	registry.SetLogger(zap.NewNop())

	// محاولة الحصول على حالة العميل البشري بدون تسجيل
	_, err := registry.GetHumanClientStatus()
	if err == nil {
		t.Error("يجب أن يفشل الحصول على حالة العميل البشري عندما لم يتم تسجيله")
	}

	t.Log("تم التحقق من خطأ عدم تسجيل العميل البشري بنجاح")
}
