package node

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/sirupsen/logrus"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// A2ANetworkBridge يربط A2AManager المحلي بالشبكة (PubSub)
// [WHY] يمكّن الوكلاء على أجهزة مختلفة من التواصل عبر A2A
// [HOW] يشترك في أحداث a2a.message و a2a.broadcast من EventBus المحلي
//       ويعيد توجيهها عبر PubSub للعقد الأخرى
type A2ANetworkBridge struct {
	node      *Node
	sessionID string
	localBus  *eventbus.EventBus
	topic     *pubsub.Topic
	sub       *pubsub.Subscription
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	running   bool
	log       *logrus.Logger
}

// A2ABridgeEvent رسالة A2A عبر الشبكة
type A2ABridgeEvent struct {
	SessionID string          `json:"session_id"`
	EventType string          `json:"event_type"`
	SourceNode string         `json:"source_node"`
	Message    json.RawMessage `json:"message"`
	Timestamp  int64           `json:"timestamp"`
}

// BridgeA2AToNetwork ينشئ جسر A2A عبر الشبكة
func (n *Node) BridgeA2AToNetwork(ctx context.Context, sessionID string, localBus *eventbus.EventBus) (*A2ANetworkBridge, error) {
	topicName := fmt.Sprintf("/mskt/session/%s/a2a", sessionID)
	topic, err := n.ps().Join(topicName)
	if err != nil {
		return nil, fmt.Errorf("فشل الانضمام لموضوع A2A %s: %w", sessionID, err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("فشل الاشتراك في A2A %s: %w", sessionID, err)
	}

	ctx, cancel := context.WithCancel(ctx)

	bridge := &A2ANetworkBridge{
		node:      n,
		sessionID: sessionID,
		localBus:  localBus,
		topic:     topic,
		sub:       sub,
		ctx:       ctx,
		cancel:    cancel,
		log:       n.log,
		running:   true,
	}

	// اشتراك في أحداث A2A المحلية
	localBus.Subscribe("a2a.message", bridge.handleLocalA2AMessage)
	localBus.Subscribe("a2a.broadcast", bridge.handleLocalA2ABroadcast)

	// استقبال أحداث A2A من الشبكة
	bridge.wg.Add(1)
	go bridge.receiveNetworkA2AEvents()

	n.log.WithField("session_id", sessionID).Info("تم إنشاء جسر A2A عبر الشبكة")
	return bridge, nil
}

// handleLocalA2AMessage يرسل رسالة A2A محلية إلى الشبكة
func (ab *A2ANetworkBridge) handleLocalA2AMessage(evt eventbus.Event) {
	ab.publishToNetwork("a2a.message", evt.Payload)
}

// handleLocalA2ABroadcast يرسل بث A2A محلي إلى الشبكة
func (ab *A2ANetworkBridge) handleLocalA2ABroadcast(evt eventbus.Event) {
	ab.publishToNetwork("a2a.broadcast", evt.Payload)
}

func (ab *A2ANetworkBridge) publishToNetwork(eventType string, payload interface{}) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}

	event := A2ABridgeEvent{
		SessionID:  ab.sessionID,
		EventType:  eventType,
		SourceNode: ab.node.host().ID().String(),
		Message:    payloadJSON,
		Timestamp:  time.Now().Unix(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	if err := ab.topic.Publish(ab.ctx, data); err != nil {
		ab.log.WithField("session_id", ab.sessionID).WithError(err).Warn("فشل نشر A2A")
	}
}

func (ab *A2ANetworkBridge) receiveNetworkA2AEvents() {
	defer ab.wg.Done()

	for {
		msg, err := ab.sub.Next(ab.ctx)
		if err != nil {
			if ab.ctx.Err() == nil {
				ab.log.WithField("session_id", ab.sessionID).WithError(err).Warn("فشل استقبال A2A")
			}
			return
		}

		if msg.GetFrom() == ab.node.host().ID() {
			continue
		}

		var bridgeEvent A2ABridgeEvent
		if err := json.Unmarshal(msg.Data, &bridgeEvent); err != nil {
			continue
		}

		// إعادة توجيه الرسالة إلى EventBus المحلي
		// A2AManager المحلي يستقبلها عبر اشتراكه في a2a.message / a2a.broadcast
		ab.localBus.Publish(eventbus.Event{
			Type:      bridgeEvent.EventType,
			Source:    bridgeEvent.SourceNode,
			SessionID: ab.sessionID,
			Payload:   bridgeEvent.Message,
			Timestamp: time.Unix(bridgeEvent.Timestamp, 0),
		})
	}
}

// Close يغلق جسر A2A
func (ab *A2ANetworkBridge) Close() error {
	ab.mu.Lock()
	if !ab.running {
		ab.mu.Unlock()
		return nil
	}
	ab.running = false
	ab.mu.Unlock()

	ab.cancel()
	ab.wg.Wait()
	return nil
}
