package channel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// ChannelManager واجهة بسيطة لإدارة القنوات (للاستخدام في الاختبارات)
type ChannelManager interface {
	Publish(ctx context.Context, channelID string, msg interface{}) error
	Subscribe(ctx context.Context, channelID string, handler func([]byte)) (interface{}, error)
}

// ThreadedChat يدير المحادثات الخاصة بعقدة معينة في سير عمل
type ThreadedChat struct {
	channelMgr ChannelManager
}

// NewThreadedChat ينشئ مدير شات سياقي جديد
func NewThreadedChat(channelMgr ChannelManager) *ThreadedChat {
	return &ThreadedChat{channelMgr: channelMgr}
}

// ThreadMessage يمثل رسالة في شات العقدة
type ThreadMessage struct {
	ID        string    `json:"id"`
	SenderDID string    `json:"sender_did"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GetChannelID يولد معرف القناة الفريد للعقدة
func (tc *ThreadedChat) GetChannelID(workflowID, nodeID string) string {
	return fmt.Sprintf("thread_wf_%s_node_%s", workflowID, nodeID)
}

// generateID يولد معرف فريد للرسالة
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// SendMessageToNode يرسل رسالة إلى شات عقدة محددة
func (tc *ThreadedChat) SendMessageToNode(ctx context.Context, workflowID, nodeID, senderDID, content string) error {
	msg := ThreadMessage{
		ID:        generateID(),
		SenderDID: senderDID,
		Content:   content,
		Timestamp: time.Now(),
	}

	channelID := tc.GetChannelID(workflowID, nodeID)
	return tc.channelMgr.Publish(ctx, channelID, msg)
}

// SubscribeToNodeThread يشترك في تلقي رسائل عقدة محددة
func (tc *ThreadedChat) SubscribeToNodeThread(ctx context.Context, workflowID, nodeID string, callback func(msg ThreadMessage)) error {
	channelID := tc.GetChannelID(workflowID, nodeID)

	_, err := tc.channelMgr.Subscribe(ctx, channelID, func(msgData []byte) {
		var msg ThreadMessage
		if err := json.Unmarshal(msgData, &msg); err != nil {
			return
		}
		callback(msg)
	})

	return err
}
