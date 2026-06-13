package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/common"
	"github.com/MortalArena/Musketeers/pkg/crypto"
)

// ChannelManager واجهة بسيطة لإدارة القنوات (للاستخدام في الاختبارات)
type ChannelManager interface {
	Publish(ctx context.Context, channelID string, msg interface{}) error
	Subscribe(ctx context.Context, channelID string, handler func([]byte)) (interface{}, error)
}

// CRDTSyncManager يدير نقل وتحقق تحديثات CRDT عبر الشبكة اللامركزية
type CRDTSyncManager struct {
	channelMgr  ChannelManager
	keyPair     *crypto.KeyPair
	documentID  string
	mu          sync.RWMutex
	subscribers map[string]func(update []byte, senderDID string)
	keyResolver common.KeyResolver
}

// NewCRDTSyncManager ينشئ مدير مزامنة لسير عمل محدد
func NewCRDTSyncManager(channelMgr ChannelManager, kp *crypto.KeyPair, documentID string, resolver common.KeyResolver) *CRDTSyncManager {
	return &CRDTSyncManager{
		channelMgr:  channelMgr,
		keyPair:     kp,
		documentID:  documentID,
		subscribers: make(map[string]func(update []byte, senderDID string)),
		keyResolver: resolver,
	}
}

// CRDTMessage يمثل رسالة تحديث CRDT الموقعة
type CRDTMessage struct {
	DocumentID string `json:"document_id"`
	Payload    []byte `json:"payload"` // Yjs Binary Update
	SenderDID  string `json:"sender_did"`
	Signature  string `json:"signature"`
}

// BroadcastUpdate يوقع ويبث تحديث CRDT إلى جميع المشاركين
func (c *CRDTSyncManager) BroadcastUpdate(ctx context.Context, payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("payload cannot be empty")
	}

	// 1. توقيع التحديث باستخدام المفتاح الخاص للمستخدم
	privKey := c.keyPair.Private

	domain := crypto.DomainDirectMsg + c.documentID + "|"
	sig, err := crypto.SignPayloadHex(privKey, domain, string(payload))
	if err != nil {
		return fmt.Errorf("failed to sign CRDT update: %w", err)
	}

	msg := CRDTMessage{
		DocumentID: c.documentID,
		Payload:    payload,
		SenderDID:  c.keyPair.DID,
		Signature:  sig,
	}

	// 2. تحديد اسم القناة الخاص بهذا المستند
	channelID := fmt.Sprintf("crdt_sync_%s", c.documentID)

	// 3. نشر الرسالة عبر القناة
	if err := c.channelMgr.Publish(ctx, channelID, msg); err != nil {
		return fmt.Errorf("failed to publish CRDT update: %w", err)
	}

	return nil
}

// Subscribe يسمح للعميل بالاستماع للتحديثات والتحقق منها
func (c *CRDTSyncManager) Subscribe(ctx context.Context, subscriberID string, callback func(update []byte, senderDID string)) error {
	c.mu.Lock()
	c.subscribers[subscriberID] = callback
	c.mu.Unlock()

	channelID := fmt.Sprintf("crdt_sync_%s", c.documentID)

	// الاشتراك في القناة
	_, err := c.channelMgr.Subscribe(ctx, channelID, func(msgData []byte) {
		var msg CRDTMessage
		if err := json.Unmarshal(msgData, &msg); err != nil {
			return
		}

		if msg.DocumentID != c.documentID {
			return // تجاهل الرسائل غير المتعلقة بهذا المستند
		}

		// 4. التحقق من صحة التوقيع قبل تسليم التحديث للتطبيق
		pubKey, err := c.keyResolver.ResolvePublicKey(msg.SenderDID)
		if err != nil {
			return // سجل تحذير أمني: فشل حل المفتاح
		}

		domain := crypto.DomainDirectMsg + c.documentID + "|"
		if err := crypto.VerifyPayloadHex(pubKey, domain, string(msg.Payload), msg.Signature); err != nil {
			return // سجل تحذير أمني: تحديث مزيف
		}

		// تسليم التحديث الآمن للمعالج
		c.mu.RLock()
		cb := c.subscribers[subscriberID]
		c.mu.RUnlock()

		if cb != nil {
			cb(msg.Payload, msg.SenderDID)
		}
	})

	return err
}

// Unsubscribe يلغي الاشتراك
func (c *CRDTSyncManager) Unsubscribe(subscriberID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscribers, subscriberID)
}
