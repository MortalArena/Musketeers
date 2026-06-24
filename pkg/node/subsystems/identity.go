package subsystems

import (
	"crypto/ed25519"
	"sync"

	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/identity"
)

type IdentitySubsystem struct {
	mu       sync.RWMutex
	keyPair  *nrcrypto.KeyPair
	identity *identity.IdentityRecord
	keyCache map[string]ed25519.PublicKey
}

func NewIdentitySubsystem(keyPair *nrcrypto.KeyPair, identity *identity.IdentityRecord) *IdentitySubsystem {
	return &IdentitySubsystem{keyPair: keyPair, identity: identity, keyCache: make(map[string]ed25519.PublicKey)}
}

func (s *IdentitySubsystem) KeyPair() *nrcrypto.KeyPair             { return s.keyPair }
func (s *IdentitySubsystem) Identity() *identity.IdentityRecord     { return s.identity }

func (s *IdentitySubsystem) CacheGet(did string) (ed25519.PublicKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pub, ok := s.keyCache[did]
	return pub, ok
}

func (s *IdentitySubsystem) CacheSet(did string, pub ed25519.PublicKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyCache[did] = pub
}

func (s *IdentitySubsystem) CacheLen() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.keyCache)
}

func (s *IdentitySubsystem) CacheClearTo(target int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.keyCache {
		delete(s.keyCache, k)
		if len(s.keyCache) <= target {
			break
		}
	}
}
