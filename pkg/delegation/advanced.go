package delegation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/common"
	"github.com/MortalArena/Musketeers/pkg/crypto"
)

// DelegationScope يحدد نطاق التفويض بدقة
type DelegationScope struct {
	WorkflowID     string   `json:"workflow_id"`
	AllowedNodeIDs []string `json:"allowed_node_ids,omitempty"` // فارغ = كل العقد
	AllowedActions []string `json:"allowed_actions"`            // e.g., "edit", "execute", "read"
}

// DelegationRecord سجل التفويض الموقع
type DelegationRecord struct {
	ID           string          `json:"id"`
	DelegatorDID string          `json:"delegator_did"`
	DelegateDID  string          `json:"delegate_did"`
	Scope        DelegationScope `json:"scope"`
	ExpiresAt    time.Time       `json:"expires_at"`
	Signature    []byte          `json:"signature"` // توقيع المفوض (Delegator)
}

// DelegationManager يدير إنشاء والتحقق من التفويضات
type DelegationManager struct {
	keyResolver common.KeyResolver // واجهة لجلب المفاتيح العامة من DID
}

// NewDelegationManager ينشئ مدير تفويض جديد
func NewDelegationManager(resolver common.KeyResolver) *DelegationManager {
	return &DelegationManager{keyResolver: resolver}
}

// generateID يولد معرف فريد للتفويض
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateDelegation ينشئ ويفوض صلاحيات جديدة
func (dm *DelegationManager) CreateDelegation(delegatorPrivKey ed25519.PrivateKey, delegateDID string, scope DelegationScope, duration time.Duration) (*DelegationRecord, error) {
	if delegatorPrivKey == nil {
		return nil, fmt.Errorf("delegator private key is required")
	}

	// حساب DID من المفتاح العام
	pubKey := delegatorPrivKey.Public().(ed25519.PublicKey)
	delegatorDID := crypto.DIDFromPublicKey(pubKey)

	record := &DelegationRecord{
		ID:           generateID(),
		DelegatorDID: delegatorDID,
		DelegateDID:  delegateDID,
		Scope:        scope,
		ExpiresAt:    time.Now().Add(duration),
	}

	// توقيع السجل
	dataToSign, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal delegation record: %w", err)
	}

	domain := crypto.DomainDelegation
	sig, err := crypto.SignPayload(delegatorPrivKey, domain, string(dataToSign))
	if err != nil {
		return nil, fmt.Errorf("failed to sign delegation record: %w", err)
	}
	record.Signature = sig

	return record, nil
}

// VerifyDelegation يتحقق من صحة التفويض وصلاحياته
func (dm *DelegationManager) VerifyDelegation(record *DelegationRecord, requestedAction string, targetNodeID string) error {
	// 1. التحقق من انتهاء الصلاحية
	if time.Now().After(record.ExpiresAt) {
		return fmt.Errorf("delegation expired")
	}

	// 2. التحقق من التوقيع الرقمي
	pubKey, err := dm.keyResolver.ResolvePublicKey(record.DelegatorDID)
	if err != nil {
		return fmt.Errorf("failed to resolve delegator public key: %w", err)
	}

	// إزالة حقل التوقيع من البيانات الموقعة للتحقق
	recordCopy := *record
	recordCopy.Signature = nil
	dataToVerify, err := json.Marshal(recordCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal record for verification: %w", err)
	}

	domain := crypto.DomainDelegation
	if err := crypto.VerifyPayload(pubKey, domain, string(dataToVerify), record.Signature); err != nil {
		return fmt.Errorf("invalid delegation signature: %w", err)
	}

	// 3. التحقق من نطاق الصلاحيات (Actions)
	actionAllowed := false
	for _, action := range record.Scope.AllowedActions {
		if action == requestedAction || action == "*" {
			actionAllowed = true
			break
		}
	}
	if !actionAllowed {
		return fmt.Errorf("action '%s' not allowed in delegation scope", requestedAction)
	}

	// 4. التحقق من نطاق العقد (Nodes)
	if len(record.Scope.AllowedNodeIDs) > 0 {
		nodeAllowed := false
		for _, nodeID := range record.Scope.AllowedNodeIDs {
			if nodeID == targetNodeID || nodeID == "*" {
				nodeAllowed = true
				break
			}
		}
		if !nodeAllowed {
			return fmt.Errorf("target node '%s' not allowed in delegation scope", targetNodeID)
		}
	}

	return nil
}

// RevokeDelegation يلغي التفويض (في التنفيذ الحقيقي، يتم تخزين التفويضات الملغاة)
func (dm *DelegationManager) RevokeDelegation(record *DelegationRecord) error {
	// في التنفيذ الحقيقي، يتم تخزين التفويض الملغى في قاعدة بيانات
	// هنا سنقوم بمحاكاة الإلغاء بتعيين تاريخ انتهاء الصلاحية إلى الماضي
	record.ExpiresAt = time.Now().Add(-1 * time.Hour)
	return nil
}
