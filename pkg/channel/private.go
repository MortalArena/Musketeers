package channel

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"filippo.io/edwards25519"
	"github.com/MortalArena/Musketeers/pkg/protocol"
	"golang.org/x/crypto/curve25519"
)

// ChannelConfig إعدادات قناة خاصة
type ChannelConfig struct {
	ID         string            `json:"id"`
	Owner      string            `json:"owner"`
	Members    []string          `json:"members"`
	Admins     []string          `json:"admins"`
	SharedKey  string            `json:"shared_key"`            // hex — مفتاح AES مشفّر بـ ECDH للمالك
	MemberKeys map[string]string `json:"member_keys,omitempty"` // DID -> مفتاح مشفّر لكل عضو
	KeyVersion uint64            `json:"key_version"`
	Signature  string            `json:"signature"`
	// [FIX] إضافة حالة الوكلاء للقناة الخاصة
	AgentStates map[string]AgentState `json:"agent_states,omitempty"` // DID -> حالة الوكيل
}

// AgentState حالة الوكيل في القناة الخاصة
type AgentState struct {
	DID         string    `json:"did"`
	Name        string    `json:"name"`
	Status      string    `json:"status"` // "idle", "busy", "offline", "available"
	LastSeen    time.Time `json:"last_seen"`
	CurrentTask string    `json:"current_task,omitempty"` // المهمة الحالية إن وجدت
	Priority    int       `json:"priority"`               // أولوية الوكيل للاتصال
}

// ChannelConfigPayload payload توقيع الإعدادات
func ChannelConfigPayload(cfg *ChannelConfig) string {
	members := ""
	for i, m := range cfg.Members {
		if i > 0 {
			members += ","
		}
		members += m
	}
	admins := ""
	for i, a := range cfg.Admins {
		if i > 0 {
			admins += ","
		}
		admins += a
	}
	return cfg.ID + "|" + cfg.Owner + "|" + members + "|" + admins + "|" + cfg.SharedKey
}

// NewPrivateChannel ينشئ قناة خاصة جديدة
func NewPrivateChannel(id, ownerDID string, ownerPriv ed25519.PrivateKey, members, admins []string) (*ChannelConfig, []byte, error) {
	// توليد مفتاح AES-256 عشوائي
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, nil, fmt.Errorf("فشل توليد مفتاح AES: %w", err)
	}

	// تشفير المفتاح بـ ECDH مع مفتاح المالك
	ownerPub := ownerPriv.Public().(ed25519.PublicKey)
	encryptedKey, err := encryptKeyECDH(aesKey, ownerPub)
	if err != nil {
		return nil, nil, err
	}

	cfg := &ChannelConfig{
		ID:         id,
		Owner:      ownerDID,
		Members:    members,
		Admins:     admins,
		SharedKey:  hex.EncodeToString(encryptedKey),
		MemberKeys: make(map[string]string),
		KeyVersion: 1,
	}
	if err := signConfig(cfg, ownerPriv); err != nil {
		return nil, nil, err
	}
	return cfg, aesKey, nil
}

// encryptKeyECDH يشفّر مفتاح AES باستخدام ECDH و AES-GCM
func encryptKeyECDH(aesKey []byte, ownerPub ed25519.PublicKey) ([]byte, error) {
	// تحويل Ed25519 إلى Curve25519
	p, err := new(edwards25519.Point).SetBytes(ownerPub)
	if err != nil {
		return nil, fmt.Errorf("مفتاح عام Ed25519 غير صالح: %w", err)
	}
	ownerCurve := p.BytesMontgomery()

	ephemeralPriv := make([]byte, 32)
	if _, err := rand.Read(ephemeralPriv); err != nil {
		return nil, err
	}
	var ephemeralPub [32]byte
	curve25519.ScalarBaseMult(&ephemeralPub, (*[32]byte)(ephemeralPriv))

	var shared [32]byte
	curve25519.ScalarMult(&shared, (*[32]byte)(ephemeralPriv), (*[32]byte)(ownerCurve))

	// تشفير AES key باستخدام AES-GCM حيث المفتاح هو الـ shared secret
	block, err := aes.NewCipher(shared[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, aesKey, nil)

	// الناتج النهائي: ephemeral public key + nonce + ciphertext
	result := make([]byte, 32+len(nonce)+len(ciphertext))
	copy(result[:32], ephemeralPub[:])
	copy(result[32:32+len(nonce)], nonce)
	copy(result[32+len(nonce):], ciphertext)
	return result, nil
}

// DecryptSharedKey يفك تشفير مفتاح القناة باستخدام AES-GCM
func DecryptSharedKey(encryptedHex string, ownerPriv ed25519.PrivateKey) ([]byte, error) {
	encrypted, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return nil, err
	}
	if len(encrypted) < 32 {
		return nil, fmt.Errorf("بيانات مشفرة قصيرة جداً")
	}

	// تحويل Ed25519 لـ Curve25519
	seed := ownerPriv.Seed()
	h := sha512.Sum512(seed)
	ownerCurve := h[:32]
	ownerCurve[0] &= 248
	ownerCurve[31] &= 127
	ownerCurve[31] |= 64

	var ephemeralPub [32]byte
	copy(ephemeralPub[:], encrypted[:32])

	var shared [32]byte
	curve25519.ScalarMult(&shared, (*[32]byte)(ownerCurve), &ephemeralPub)

	block, err := aes.NewCipher(shared[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(encrypted) < 32+nonceSize {
		return nil, fmt.Errorf("بيانات غير كافية للـ nonce")
	}

	nonce := encrypted[32 : 32+nonceSize]
	ciphertext := encrypted[32+nonceSize:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// EncryptPrivateMessage يشفّر رسالة خاصة بـ AES-256-GCM
func EncryptPrivateMessage(channelID string, plaintext *protocol.PrivatePlaintext, aesKey []byte) (*protocol.EncryptedMessage, error) {
	plainJSON, err := json.Marshal(plaintext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// AAD = channelID
	ciphertext := gcm.Seal(nil, nonce, plainJSON, []byte(channelID))

	return &protocol.EncryptedMessage{
		Nonce:      hex.EncodeToString(nonce),
		Ciphertext: hex.EncodeToString(ciphertext),
	}, nil
}

// DecryptPrivateMessage يفك تشفير رسالة خاصة
func DecryptPrivateMessage(channelID string, enc *protocol.EncryptedMessage, aesKey []byte) (*protocol.PrivatePlaintext, error) {
	nonce, err := hex.DecodeString(enc.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := hex.DecodeString(enc.Ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plainJSON, err := gcm.Open(nil, nonce, ciphertext, []byte(channelID))
	if err != nil {
		return nil, fmt.Errorf("فشل فك التشفير: %w", err)
	}

	var plaintext protocol.PrivatePlaintext
	if err := json.Unmarshal(plainJSON, &plaintext); err != nil {
		return nil, err
	}
	return &plaintext, nil
}

// IsMember يتحقق من عضوية DID
func (cfg *ChannelConfig) IsMember(did string) bool {
	if cfg.Owner == did {
		return true
	}
	for _, m := range cfg.Members {
		if m == did {
			return true
		}
	}
	return false
}

// IsAdmin يتحقق من صلاحية المشرف
func (cfg *ChannelConfig) IsAdmin(did string) bool {
	if cfg.Owner == did {
		return true
	}
	for _, a := range cfg.Admins {
		if a == did {
			return true
		}
	}
	return false
}

// VerifyConfig يتحقق من توقيع إعدادات القناة
func (cfg *ChannelConfig) Verify(ownerPub ed25519.PublicKey) error {
	return cfg.VerifyConfigV2(ownerPub)
}

// ============================================================
// [FIX] إدارة حالة الوكلاء في القنوات الخاصة
// ============================================================

// UpdateAgentState يحدث حالة وكيل في القناة الخاصة
func (cfg *ChannelConfig) UpdateAgentState(agentDID, name, status, currentTask string, priority int) {
	if cfg.AgentStates == nil {
		cfg.AgentStates = make(map[string]AgentState)
	}

	cfg.AgentStates[agentDID] = AgentState{
		DID:         agentDID,
		Name:        name,
		Status:      status,
		LastSeen:    time.Now(),
		CurrentTask: currentTask,
		Priority:    priority,
	}
}

// GetAgentState يحصل على حالة وكيل محدد
func (cfg *ChannelConfig) GetAgentState(agentDID string) (AgentState, bool) {
	if cfg.AgentStates == nil {
		return AgentState{}, false
	}
	state, ok := cfg.AgentStates[agentDID]
	return state, ok
}

// GetAvailableAgents يحصل على قائمة الوكلاء المتاحين (idle أو available)
func (cfg *ChannelConfig) GetAvailableAgents() []AgentState {
	if cfg.AgentStates == nil {
		return []AgentState{}
	}

	var available []AgentState
	for _, state := range cfg.AgentStates {
		if state.Status == "idle" || state.Status == "available" {
			available = append(available, state)
		}
	}

	// ترتيب حسب الأولوية (الأعلى أولاً)
	for i := 0; i < len(available); i++ {
		for j := i + 1; j < len(available); j++ {
			if available[j].Priority > available[i].Priority {
				available[i], available[j] = available[j], available[i]
			}
		}
	}

	return available
}

// GetBusyAgents يحصل على قائمة الوكلاء المشغولين
func (cfg *ChannelConfig) GetBusyAgents() []AgentState {
	if cfg.AgentStates == nil {
		return []AgentState{}
	}

	var busy []AgentState
	for _, state := range cfg.AgentStates {
		if state.Status == "busy" {
			busy = append(busy, state)
		}
	}

	return busy
}

// GetAllAgentStates يحصل على جميع حالات الوكلاء
func (cfg *ChannelConfig) GetAllAgentStates() []AgentState {
	if cfg.AgentStates == nil {
		return []AgentState{}
	}

	states := make([]AgentState, 0, len(cfg.AgentStates))
	for _, state := range cfg.AgentStates {
		states = append(states, state)
	}

	return states
}
