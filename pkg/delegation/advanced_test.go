package delegation

import (
	"crypto/ed25519"
	"testing"
	"time"
)

// MockKeyResolver للتجربة
type MockDelegationKeyResolver struct {
	pubKey ed25519.PublicKey
}

func (m *MockDelegationKeyResolver) ResolvePublicKey(did string) (ed25519.PublicKey, error) {
	return m.pubKey, nil
}

func TestDelegationManager_CreateDelegation(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1", "node2"},
		AllowedActions: []string{"edit", "execute"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	if record == nil {
		t.Fatal("Expected record to be non-nil")
	}

	if record.ID == "" {
		t.Error("Expected record ID to be non-empty")
	}

	if record.DelegatorDID == "" {
		t.Error("Expected delegator DID to be non-empty")
	}

	if record.DelegateDID != "did:mskt:delegate" {
		t.Errorf("Expected delegate DID did:mskt:delegate, got %s", record.DelegateDID)
	}

	if record.Signature == nil {
		t.Error("Expected signature to be non-nil")
	}

	if time.Now().After(record.ExpiresAt) {
		t.Error("Expected delegation to not be expired immediately")
	}
}

func TestDelegationManager_CreateDelegation_NilKey(t *testing.T) {
	mockResolver := &MockDelegationKeyResolver{}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"edit"},
	}

	_, err := dm.CreateDelegation(nil, "did:mskt:delegate", scope, 24*time.Hour)
	if err == nil {
		t.Error("Expected error for nil private key")
	}
}

func TestDelegationManager_VerifyDelegation_Success(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"edit", "execute"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	err = dm.VerifyDelegation(record, "edit", "node1")
	if err != nil {
		t.Errorf("VerifyDelegation failed: %v", err)
	}
}

func TestDelegationManager_VerifyDelegation_Expired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"edit"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, -1*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	err = dm.VerifyDelegation(record, "edit", "node1")
	if err == nil {
		t.Error("Expected error for expired delegation")
	}
}

func TestDelegationManager_VerifyDelegation_ActionNotAllowed(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"read"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	err = dm.VerifyDelegation(record, "edit", "node1")
	if err == nil {
		t.Error("Expected error for action not allowed")
	}
}

func TestDelegationManager_VerifyDelegation_NodeNotAllowed(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"edit"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	err = dm.VerifyDelegation(record, "edit", "node2")
	if err == nil {
		t.Error("Expected error for node not allowed")
	}
}

func TestDelegationManager_VerifyDelegation_WildcardAction(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"*"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	err = dm.VerifyDelegation(record, "any_action", "node1")
	if err != nil {
		t.Errorf("VerifyDelegation with wildcard action failed: %v", err)
	}
}

func TestDelegationManager_VerifyDelegation_WildcardNode(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"*"},
		AllowedActions: []string{"edit"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	err = dm.VerifyDelegation(record, "edit", "any_node")
	if err != nil {
		t.Errorf("VerifyDelegation with wildcard node failed: %v", err)
	}
}

func TestDelegationManager_VerifyDelegation_EmptyNodeIDs(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{}, // فارغ = كل العقد
		AllowedActions: []string{"edit"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	err = dm.VerifyDelegation(record, "edit", "any_node")
	if err != nil {
		t.Errorf("VerifyDelegation with empty node IDs failed: %v", err)
	}
}

func TestDelegationManager_RevokeDelegation(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"edit"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	err = dm.RevokeDelegation(record)
	if err != nil {
		t.Fatalf("RevokeDelegation failed: %v", err)
	}

	// التحقق من أن التفويض منتهي الصلاحية
	if !time.Now().After(record.ExpiresAt) {
		t.Error("Expected delegation to be expired after revocation")
	}
}

func TestDelegationManager_VerifyDelegation_InvalidSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"edit"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	// تغيير التوقيع
	record.Signature = []byte("invalid_signature")

	err = dm.VerifyDelegation(record, "edit", "node1")
	if err == nil {
		t.Error("Expected error for invalid signature")
	}
}

func TestDelegationManager_VerifyDelegation_KeyResolverError(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Mock resolver that returns error
	errorResolver := &MockDelegationKeyResolver{}
	dm := NewDelegationManager(errorResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"edit"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	err = dm.VerifyDelegation(record, "edit", "node1")
	if err == nil {
		t.Error("Expected error when key resolver fails")
	}
}

func TestDelegationManager_CreateDelegation_ZeroDuration(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"edit"},
	}

	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, -1*time.Hour)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	// التحقق من أن التفويض منتهي الصلاحية
	if !time.Now().After(record.ExpiresAt) {
		t.Error("Expected delegation to be expired")
	}
}

func TestDelegationManager_CreateDelegation_LongDuration(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	mockResolver := &MockDelegationKeyResolver{pubKey: pub}
	dm := NewDelegationManager(mockResolver)

	scope := DelegationScope{
		WorkflowID:     "workflow_123",
		AllowedNodeIDs: []string{"node1"},
		AllowedActions: []string{"edit"},
	}

	duration := 365 * 24 * time.Hour // سنة واحدة
	record, err := dm.CreateDelegation(priv, "did:mskt:delegate", scope, duration)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	expectedExpiry := time.Now().Add(duration)
	diff := expectedExpiry.Sub(record.ExpiresAt)
	if diff > 1*time.Second || diff < -1*time.Second {
		t.Errorf("Expected expiry time to be approximately %v from now, got %v", duration, time.Until(record.ExpiresAt))
	}
}
