package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// UserState يمثل الحالة الحالية لمستخدم أو وكيل في المستند
type UserState struct {
	DID            string    `json:"did"`
	Name           string    `json:"name"`
	Color          string    `json:"color"`
	CursorPosition []float64 `json:"cursor_position,omitempty"` // [x, y]
	SelectedNodes  []string  `json:"selected_nodes,omitempty"`
	LastSeen       time.Time `json:"last_seen"`
}

// PresenceManager يدير حالات الحضور ويوزعها
type PresenceManager struct {
	channelMgr ChannelManager
	documentID string
	localState UserState

	mu           sync.RWMutex
	remoteStates map[string]UserState // DID -> UserState
	subscribers  map[string]func(states map[string]UserState)

	stopChan chan struct{}
}

// NewPresenceManager ينشئ مدير حضور جديد
func NewPresenceManager(channelMgr ChannelManager, documentID string, initialDID, initialName string) *PresenceManager {
	pm := &PresenceManager{
		channelMgr:   channelMgr,
		documentID:   documentID,
		remoteStates: make(map[string]UserState),
		subscribers:  make(map[string]func(states map[string]UserState)),
		stopChan:     make(chan struct{}),
		localState: UserState{
			DID:      initialDID,
			Name:     initialName,
			Color:    "#000000", // يمكن تخصيصه لاحقاً
			LastSeen: time.Now(),
		},
	}

	go pm.cleanupRoutine()
	return pm
}

// UpdateLocalState يحدث حالة المستخدم المحلي ويبثها للآخرين
func (pm *PresenceManager) UpdateLocalState(cursor []float64, selectedNodes []string) error {
	pm.mu.Lock()
	pm.localState.CursorPosition = cursor
	pm.localState.SelectedNodes = selectedNodes
	pm.localState.LastSeen = time.Now()

	// نسخة عميقة للبث
	stateToBroadcast := pm.localState
	pm.mu.Unlock()

	channelID := fmt.Sprintf("presence_%s", pm.documentID)
	return pm.channelMgr.Publish(context.Background(), channelID, stateToBroadcast)
}

// Subscribe يستمع لحالات الآخرين
func (pm *PresenceManager) Subscribe(subscriberID string, callback func(states map[string]UserState)) error {
	pm.mu.Lock()
	pm.subscribers[subscriberID] = callback
	pm.mu.Unlock()

	channelID := fmt.Sprintf("presence_%s", pm.documentID)
	_, err := pm.channelMgr.Subscribe(context.Background(), channelID, func(msgData []byte) {
		var state UserState
		if err := json.Unmarshal(msgData, &state); err != nil {
			return
		}

		pm.mu.Lock()
		state.LastSeen = time.Now() // تحديث وقت آخر رؤية
		pm.remoteStates[state.DID] = state

		// نسخ الحالات لإرسالها للمعالجين
		statesCopy := make(map[string]UserState)
		for k, v := range pm.remoteStates {
			statesCopy[k] = v
		}
		pm.mu.Unlock()

		pm.mu.RLock()
		cb := pm.subscribers[subscriberID]
		pm.mu.RUnlock()

		if cb != nil {
			cb(statesCopy)
		}
	})

	return err
}

// cleanupRoutine يزيل المستخدمين غير النشطين كل 30 ثانية
func (pm *PresenceManager) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.mu.Lock()
			now := time.Now()
			for did, state := range pm.remoteStates {
				if now.Sub(state.LastSeen) > 60*time.Second {
					delete(pm.remoteStates, did)

					// إشعار المشتركين بالتغيير
					statesCopy := make(map[string]UserState)
					for k, v := range pm.remoteStates {
						statesCopy[k] = v
					}
					for _, cb := range pm.subscribers {
						go cb(statesCopy)
					}
				}
			}
			pm.mu.Unlock()
		}
	}
}

// Close يوقف المدير وينظف الموارد
func (pm *PresenceManager) Close() {
	close(pm.stopChan)
}

// GetRemoteStates يرجع نسخة من حالات المستخدمين البعيدين
func (pm *PresenceManager) GetRemoteStates() map[string]UserState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	statesCopy := make(map[string]UserState)
	for k, v := range pm.remoteStates {
		statesCopy[k] = v
	}
	return statesCopy
}

// GetLocalState يرجع نسخة من حالة المستخدم المحلي
func (pm *PresenceManager) GetLocalState() UserState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.localState
}
