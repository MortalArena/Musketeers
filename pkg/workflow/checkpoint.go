package workflow

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/content"
)

// Checkpoint يمثل لقطة لحالة سير العمل في نقطة زمنية محددة
type Checkpoint struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	NodeID     string                 `json:"node_id"` // آخر عقدة تم إكمالها بنجاح
	State      map[string]interface{} `json:"state"`   // حالة المتغيرات والبيانات
	Hash       string                 `json:"hash"`    // للتأكد من عدم التلاعب
	Timestamp  time.Time              `json:"timestamp"`
}

// PolicyEngine واجهة للتحقق من الصلاحيات
type PolicyEngine interface {
	Check(did, resource, action, target string) bool
}

// CheckpointManager يدير عمليات حفظ واستعادة النقاط
type CheckpointManager struct {
	store        content.BlockStore
	policyEngine PolicyEngine
}

// NewCheckpointManager ينشئ مدير نقاط حفظ جديد
func NewCheckpointManager(store content.BlockStore, policyEngine PolicyEngine) *CheckpointManager {
	return &CheckpointManager{store: store, policyEngine: policyEngine}
}

// Save يحفظ حالة سير العمل بشكل آمن
func (cm *CheckpointManager) Save(workflowID, nodeID string, state map[string]interface{}, did string) error {
	cp := &Checkpoint{
		ID:         generateID(),
		WorkflowID: workflowID,
		NodeID:     nodeID,
		State:      state,
		Timestamp:  time.Now(),
	}

	// 1. حساب Hash للحالة لمنع التلاعب
	stateBytes, err := json.Marshal(cp.State)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	hash := sha256.Sum256(stateBytes)
	cp.Hash = hex.EncodeToString(hash[:])

	// 2. حفظ النقطة
	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	cid := content.CIDFromData(data)
	if err := cm.store.Put(cid, data, did); err != nil {
		return fmt.Errorf("failed to store checkpoint: %w", err)
	}

	// 3. تحديث مؤشر "آخر نقطة حفظ" لهذا الـ Workflow
	lastData := []byte(cid)
	lastCID := content.CIDFromData(lastData)
	if err := cm.store.Put(lastCID, lastData, did); err != nil {
		return fmt.Errorf("failed to update latest checkpoint pointer: %w", err)
	}

	return nil
}

// GetLatest يسترجع آخر حالة محفوظة بنجاح
func (cm *CheckpointManager) GetLatest(workflowID string, did string) (*Checkpoint, error) {
	// التحقق من الصلاحيات: هل هذا المستخدم يملك حق قراءة هذا workflow؟
	if !cm.policyEngine.Check(did, "workflow", "read", workflowID) {
		return nil, fmt.Errorf("permission denied: cannot read workflow %s", workflowID)
	}

	// 1. جلب مؤشر آخر checkpoint
	lastKey := fmt.Sprintf("checkpoint:latest:%s", workflowID)
	lastCIDBytes, err := cm.store.Get(lastKey)
	if err != nil {
		return nil, fmt.Errorf("no checkpoints found for workflow %s: %w", workflowID, err)
	}

	// 2. جلب الـ checkpoint الفعلي
	data, err := cm.store.Get(string(lastCIDBytes))
	if err != nil {
		return nil, fmt.Errorf("checkpoint data not found: %w", err)
	}

	// 3. فك التشفير
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	// 4. التحقق من سلامة البيانات
	stateBytes, _ := json.Marshal(cp.State)
	hash := sha256.Sum256(stateBytes)
	if hex.EncodeToString(hash[:]) != cp.Hash {
		return nil, fmt.Errorf("checkpoint integrity check failed")
	}

	return &cp, nil
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
