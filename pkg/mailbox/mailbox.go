package mailbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MortalArena/Musketeers/pkg/content"
	"filippo.io/edwards25519"
	"golang.org/x/crypto/curve25519"
)

type Message struct {
	ID               string    `json:"id"`
	SenderDID        string    `json:"sender_did"`
	RecipientDID     string    `json:"recipient_did"`
	EncryptedPayload []byte    `json:"encrypted_payload"`
	Nonce            []byte    `json:"nonce"`
	Timestamp        time.Time `json:"timestamp"`
}

type Mailbox struct {
	store content.BlockStore
}

func NewMailbox(store content.BlockStore) *Mailbox {
	return &Mailbox{store: store}
}

func (m *Mailbox) Send(senderDID, recipientDID string, plaintext []byte, senderPriv ed25519.PrivateKey, recipientPub ed25519.PublicKey) error {
	if len(recipientPub) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid recipient public key size: %d", len(recipientPub))
	}

	sharedSecret := deriveSharedSecret(senderPriv, recipientPub)
	aesKey := sha256.Sum256(sharedSecret)

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	encryptedPayload := gcm.Seal(nil, nonce, plaintext, nil)

	msg := &Message{
		ID:               generateID(),
		SenderDID:        senderDID,
		RecipientDID:     recipientDID,
		EncryptedPayload: encryptedPayload,
		Nonce:            nonce,
		Timestamp:        time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	cid := content.CIDFromData(data)
	if err := m.store.Put(cid, data, senderDID); err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	return nil
}

func (m *Mailbox) Fetch(recipientDID string, recipientPriv ed25519.PrivateKey) ([]*Message, error) {
	keys, err := m.store.ListKeys("")
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	var messages []*Message
	for _, key := range keys {
		data, err := m.store.Get(key)
		if err != nil {
			continue
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		if msg.RecipientDID == recipientDID {
			messages = append(messages, &msg)
		}
	}

	return messages, nil
}

func (m *Mailbox) DecryptMessage(msg *Message, recipientPriv ed25519.PrivateKey) ([]byte, error) {
	senderPub, err := resolvePublicKey(msg.SenderDID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sender public key: %w", err)
	}

	sharedSecret := deriveSharedSecret(recipientPriv, senderPub)
	aesKey := sha256.Sum256(sharedSecret)

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := msg.Nonce
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid nonce size: %d", len(nonce))
	}
	ciphertext := msg.EncryptedPayload

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func deriveSharedSecret(priv ed25519.PrivateKey, pub ed25519.PublicKey) []byte {
	curvePriv := ed25519PrivToCurve25519(priv)
	curvePub := ed25519PubToCurve25519(pub)
	secret, err := curve25519.X25519(curvePriv[:], curvePub[:])
	if err != nil {
		panic("curve25519.X25519 failed: " + err.Error())
	}
	return secret
}

func ed25519PrivToCurve25519(priv ed25519.PrivateKey) [32]byte {
	var curve [32]byte
	seed := priv.Seed()
	h := sha512.Sum512(seed)
	copy(curve[:], h[:32])
	curve[0] &= 248
	curve[31] &= 127
	curve[31] |= 64
	return curve
}

func ed25519PubToCurve25519(pub ed25519.PublicKey) [32]byte {
	var curve [32]byte
	p, err := new(edwards25519.Point).SetBytes(pub)
	if err != nil {
		panic("edwards25519.Point.SetBytes failed: " + err.Error())
	}
	copy(curve[:], p.BytesMontgomery())
	return curve
}

var resolvePublicKey = func(did string) (ed25519.PublicKey, error) {
	return nil, fmt.Errorf("key resolution not available: %s", did)
}

func SetKeyResolver(resolver func(string) (ed25519.PublicKey, error)) {
	resolvePublicKey = resolver
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString(make([]byte, 16))
	}
	return hex.EncodeToString(b)
}
