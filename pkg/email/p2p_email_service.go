package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// P2PEmailService خدمة الإيميل عبر P2P
type P2PEmailService struct {
	p2pHost    host.Host
	emailStore *EmailStore
	userEmail  string
	mu         sync.RWMutex
}

// NewP2PEmailService ينشئ خدمة إيميل جديدة
func NewP2PEmailService(p2pHost host.Host, emailStore *EmailStore, userEmail string) *P2PEmailService {
	service := &P2PEmailService{
		p2pHost:    p2pHost,
		emailStore: emailStore,
		userEmail:  userEmail,
	}

	// تسجيل handler لاستقبال الإيميلات
	p2pHost.SetStreamHandler(protocol.ID("/musketeers/email/1.0.0"), service.handleIncomingEmail)

	return service
}

// SendEmail يرسل إيميل عبر P2P
func (s *P2PEmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	// إنشاء كائن الإيميل
	email := &Email{
		ID:        generateID(),
		From:      s.userEmail,
		To:        to,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now(),
		Encrypted: true,
	}

	// حل عنوان المستلم إلى Peer ID
	recipientPeerID, err := s.resolveEmail(to)
	if err != nil {
		return fmt.Errorf("failed to resolve recipient: %w", err)
	}

	// فتح stream إلى المستلم
	stream, err := s.p2pHost.NewStream(ctx, recipientPeerID, "/musketeers/email/1.0.0")
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// تشفير الإيميل
	encryptedData, err := s.encryptEmail(email)
	if err != nil {
		return fmt.Errorf("failed to encrypt email: %w", err)
	}

	// إرسال الإيميل
	_, err = stream.Write(encryptedData)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// حفظ في Sent
	return s.emailStore.SaveSent(email)
}

// handleIncomingEmail يعالج الإيميلات الواردة
func (s *P2PEmailService) handleIncomingEmail(stream network.Stream) {
	defer stream.Close()

	// قراءة البيانات
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, err := stream.Read(buf)
	if err != nil {
		log.Printf("Failed to read email: %v", err)
		return
	}

	// فك التشفير
	email, err := s.decryptEmail(buf[:n])
	if err != nil {
		log.Printf("Failed to decrypt email: %v", err)
		return
	}

	// التحقق من أن الإيميل موجه لنا
	if email.To != s.userEmail {
		log.Printf("Email not for us: %s", email.To)
		return
	}

	// حفظ الإيميل
	if err := s.emailStore.SaveInbox(email); err != nil {
		log.Printf("Failed to save email: %v", err)
		return
	}

	log.Printf("Received email from %s: %s", email.From, email.Subject)
}

// GetInbox يحصل على صندوق الوارد
func (s *P2PEmailService) GetInbox() ([]*Email, error) {
	return s.emailStore.GetInbox(s.userEmail)
}

// GetSent يحصل على الرسائل المرسلة
func (s *P2PEmailService) GetSent() ([]*Email, error) {
	return s.emailStore.GetSent(s.userEmail)
}

// MarkAsRead يحدد إيميل كمقروء
func (s *P2PEmailService) MarkAsRead(emailID string) error {
	return s.emailStore.MarkAsRead(emailID)
}

// DeleteEmail يحذف إيميل
func (s *P2PEmailService) DeleteEmail(emailID string) error {
	return s.emailStore.Delete(emailID)
}

// resolveEmail يحل عنوان إيميل إلى Peer ID
func (s *P2PEmailService) resolveEmail(email string) (peer.ID, error) {
	// في الإنتاج، يجب استخدام DHT أو نظام أسماء
	// هذا مثال مبسط

	// البحث في Peerstore
	peers := s.p2pHost.Network().Peerstore().Peers()
	for _, p := range peers {
		storedEmail, err := s.p2pHost.Peerstore().Get(p, "email")
		if err == nil && storedEmail == email {
			return p, nil
		}
	}

	return "", fmt.Errorf("peer not found for email: %s", email)
}

// encryptEmail يشفر الإيميل
func (s *P2PEmailService) encryptEmail(email *Email) ([]byte, error) {
	// في الإنتاج، يجب استخدام تشفير حقيقي (AES, RSA, etc.)
	// هذا مثال مبسط يستخدم JSON

	data, err := json.Marshal(email)
	if err != nil {
		return nil, err
	}

	// هنا يجب إضافة تشفير حقيقي
	// encrypted := encrypt(data, recipientPublicKey)

	return data, nil
}

// decryptEmail يفك تشفير الإيميل
func (s *P2PEmailService) decryptEmail(data []byte) (*Email, error) {
	// في الإنتاج، يجب فك التشفير أولاً
	// decrypted := decrypt(data, privateKey)

	var email Email
	if err := json.Unmarshal(data, &email); err != nil {
		return nil, err
	}

	return &email, nil
}

// generateID يولد معرف فريد
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
