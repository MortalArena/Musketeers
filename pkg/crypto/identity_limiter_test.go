package crypto

import (
	"testing"
	"time"
)

func TestIdentityLimiterHumanIdentities(t *testing.T) {
	limiter := NewIdentityLimiter()

	for i := 0; i < 8; i++ {
		nodeID := "node-" + string(rune('a'+i))
		err := limiter.TryCreateIdentity(nodeID, IdentityTypeHuman)
		if err != nil {
			t.Fatalf("Failed to create human identity %d: %v", i, err)
		}
	}

	err := limiter.TryCreateIdentity("node-8", IdentityTypeHuman)
	if err == nil {
		t.Error("Should fail when human identity limit is reached")
	}

	count := limiter.GetIdentityCount(IdentityTypeHuman)
	if count != 8 {
		t.Errorf("Expected 8 human identities, got %d", count)
	}
}

func TestIdentityLimiterAgentIdentities(t *testing.T) {
	limiter := NewIdentityLimiter()

	for i := 0; i < 128; i++ {
		nodeID := "agent-node-" + string(rune('a'+i%26)) + "-" + string(rune('0'+i%10))
		err := limiter.TryCreateIdentity(nodeID, IdentityTypeAgent)
		if err != nil {
			t.Fatalf("Failed to create agent identity %d: %v", i, err)
		}
	}

	err := limiter.TryCreateIdentity("agent-node-128", IdentityTypeAgent)
	if err == nil {
		t.Error("Should fail when agent identity limit is reached")
	}

	count := limiter.GetIdentityCount(IdentityTypeAgent)
	if count != 128 {
		t.Errorf("Expected 128 agent identities, got %d", count)
	}
}

func TestIdentityLimiterCooldown(t *testing.T) {
	limiter := NewIdentityLimiter()
	limiter.SetLimits(10, 10)
	limiter.SetCooldown(1*time.Second, 1*time.Second)

	nodeID := "node-cooldown"

	err := limiter.TryCreateIdentity(nodeID, IdentityTypeHuman)
	if err != nil {
		t.Fatalf("Failed to create first human identity: %v", err)
	}

	err = limiter.TryCreateIdentity(nodeID, IdentityTypeHuman)
	if err == nil {
		t.Error("Should fail due to cooldown")
	}

	time.Sleep(1100 * time.Millisecond)

	err = limiter.TryCreateIdentity(nodeID, IdentityTypeHuman)
	if err != nil {
		t.Errorf("Should succeed after cooldown: %v", err)
	}
}

func TestIdentityLimiterTryCreateAtomic(t *testing.T) {
	limiter := NewIdentityLimiter()

	err := limiter.TryCreateIdentity("node-atomic", IdentityTypeHuman)
	if err != nil {
		t.Fatalf("TryCreateIdentity failed: %v", err)
	}

	count := limiter.GetIdentityCount(IdentityTypeHuman)
	if count != 1 {
		t.Errorf("Expected 1 identity, got %d", count)
	}
}

func TestIdentityLimiterSeparateTypes(t *testing.T) {
	limiter := NewIdentityLimiter()

	for i := 0; i < 8; i++ {
		nodeID := "human-node-" + string(rune('a'+i))
		err := limiter.TryCreateIdentity(nodeID, IdentityTypeHuman)
		if err != nil {
			t.Fatalf("Failed to create human identity %d: %v", i, err)
		}
	}

	err := limiter.TryCreateIdentity("human-node-8", IdentityTypeHuman)
	if err == nil {
		t.Error("Should fail when human identity limit is reached")
	}

	err = limiter.TryCreateIdentity("agent-node-1", IdentityTypeAgent)
	if err != nil {
		t.Errorf("Should succeed for agent identity: %v", err)
	}
}

func TestIdentityLimiterMultipleNodes(t *testing.T) {
	limiter := NewIdentityLimiter()

	for i := 0; i < 8; i++ {
		nodeID := "node-" + string(rune('a'+i))
		err := limiter.TryCreateIdentity(nodeID, IdentityTypeHuman)
		if err != nil {
			t.Fatalf("Failed to create human identity for node %s: %v", nodeID, err)
		}
	}

	count := limiter.GetIdentityCount(IdentityTypeHuman)
	if count != 8 {
		t.Errorf("Expected 8 human identities across nodes, got %d", count)
	}
}

func TestIdentityLimiterClear(t *testing.T) {
	limiter := NewIdentityLimiter()

	for i := 0; i < 5; i++ {
		nodeID := "clear-node-" + string(rune('a'+i))
		err := limiter.TryCreateIdentity(nodeID, IdentityTypeHuman)
		if err != nil {
			t.Fatalf("Failed to create identity: %v", err)
		}
	}

	limiter.Clear()

	count := limiter.GetIdentityCount(IdentityTypeHuman)
	if count != 0 {
		t.Errorf("Expected 0 identities after clear, got %d", count)
	}

	err := limiter.TryCreateIdentity("clear-node-new", IdentityTypeHuman)
	if err != nil {
		t.Errorf("Should succeed after clear: %v", err)
	}
}

func TestIdentityLimiterGetLimits(t *testing.T) {
	limiter := NewIdentityLimiter()

	maxHuman, maxAgent := limiter.GetLimits()

	if maxHuman != 8 {
		t.Errorf("Expected max human limit 8, got %d", maxHuman)
	}
	if maxAgent != 128 {
		t.Errorf("Expected max agent limit 128, got %d", maxAgent)
	}
}

func TestIdentityLimiterSetLimits(t *testing.T) {
	limiter := NewIdentityLimiter()

	limiter.SetLimits(16, 128)

	maxHuman, maxAgent := limiter.GetLimits()

	if maxHuman != 16 {
		t.Errorf("Expected max human limit 16, got %d", maxHuman)
	}
	if maxAgent != 128 {
		t.Errorf("Expected max agent limit 128, got %d", maxAgent)
	}
}
