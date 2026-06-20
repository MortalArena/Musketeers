package email

import (
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/orchestrator"
)

// ============================================================
// Email Integration - تكامل نظام البريد الإلكتروني
// ============================================================

// EmailIntegrator يربط حزمة البريد الإلكتروني مع نظام البريد الإلكتروني في orchestrator
type EmailIntegrator struct {
	emailClient  *EmailClient
	emailManager *orchestrator.EmailManager
}

// NewEmailIntegrator إنشاء مُكامل البريد الإلكتروني
func NewEmailIntegrator(config *EmailConfig, emailManager *orchestrator.EmailManager) *EmailIntegrator {
	return &EmailIntegrator{
		emailClient:  NewEmailClient(config),
		emailManager: emailManager,
	}
}

// SendViaClient إرسال بريد إلكتروني عبر عميل البريد الإلكتروني
func (ei *EmailIntegrator) SendViaClient(msg *EmailMessage) error {
	// التحقق من صحة الرسالة
	if err := ei.emailClient.Validate(msg); err != nil {
		return fmt.Errorf("email validation failed: %w", err)
	}

	// إرسال عبر عميل البريد الإلكتروني
	return ei.emailClient.Send(msg)
}

// SendViaManager إرسال بريد إلكتروني عبر مدير البريد الإلكتروني في orchestrator
func (ei *EmailIntegrator) SendViaManager(msg *EmailMessage) error {
	// تحويل رسالة البريد الإلكتروني إلى صيغة orchestrator
	orchestratorEmail := &orchestrator.Email{
		From:     msg.From,
		To:       msg.To,
		CC:       msg.CC,
		BCC:      msg.BCC,
		Subject:  msg.Subject,
		Body:     msg.Body,
		Priority: msg.Priority,
		Status:   "sent",
	}

	// إرسال عبر مدير البريد الإلكتروني
	return ei.emailManager.SendEmail(orchestratorEmail)
}

// SyncMessages مزامنة الرسائل بين النظامين
func (ei *EmailIntegrator) SyncMessages() error {
	// ملاحظة: EmailManager ليس لديه طريقة GetAllEmails حالياً
	// هذه الوظيفة ستكون متاحة بعد إضافة هذه الطريقة إلى EmailManager
	// في الوقت الحالي، يمكن مزامنة الرسائل يدوياً عبر SendViaManager
	return fmt.Errorf("GetAllEmails method not yet implemented in EmailManager")
}

// BridgeEvents جسر الأحداث بين النظامين
func (ei *EmailIntegrator) BridgeEvents() error {
	// الاستماع لأحداث البريد الإلكتروني من orchestrator
	// (تنفيذ مبسط للإيضاح)
	return nil
}
