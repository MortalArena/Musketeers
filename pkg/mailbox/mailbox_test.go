package mailbox

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/MortalArena/Musketeers/pkg/content"
	"github.com/MortalArena/Musketeers/pkg/storage"
)

func TestMailbox_SendAndFetchRoundtrip(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	mb := NewMailbox(store)

	senderPub, senderPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recipientPub, recipientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	senderDID := "did:mskt:sender"
	recipientDID := "did:mskt:recipient"
	plaintext := []byte("رسالة سرية للغاية")

	SetKeyResolver(func(did string) (ed25519.PublicKey, error) {
		if did == senderDID {
			return senderPub, nil
		}
		return nil, nil
	})

	err = mb.Send(senderDID, recipientDID, plaintext, senderPriv, recipientPub)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	msgs, err := mb.Fetch(recipientDID, recipientPriv)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	decrypted, err := mb.DecryptMessage(msgs[0], recipientPriv)
	if err != nil {
		t.Fatalf("DecryptMessage failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Decrypted text mismatch: got %s, want %s", decrypted, plaintext)
	}

	if msgs[0].SenderDID != senderDID {
		t.Errorf("SenderDID mismatch: got %s, want %s", msgs[0].SenderDID, senderDID)
	}
	if msgs[0].RecipientDID != recipientDID {
		t.Errorf("RecipientDID mismatch: got %s, want %s", msgs[0].RecipientDID, recipientDID)
	}
}

func TestMailbox_Send_EmptyRecipientKey(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	mb := NewMailbox(store)

	_, senderPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	err = mb.Send("did:mskt:sender", "did:mskt:recipient", []byte("test"), senderPriv, nil)
	if err == nil {
		t.Error("Expected error for nil recipient public key")
	}

	err = mb.Send("did:mskt:sender", "did:mskt:recipient", []byte("test"), senderPriv, ed25519.PublicKey{})
	if err == nil {
		t.Error("Expected error for empty recipient public key")
	}
}

func TestMailbox_EmptyPlaintext(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	mb := NewMailbox(store)

	_, senderPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recipientPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	err = mb.Send("did:mskt:sender", "did:mskt:recipient", []byte{}, senderPriv, recipientPub)
	if err != nil {
		t.Fatalf("Send with empty plaintext failed: %v", err)
	}
}

func TestMailbox_Fetch_Empty_DID(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	mb := NewMailbox(store)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	msgs, err := mb.Fetch("", priv)
	if err != nil {
		t.Fatalf("Fetch with empty DID failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(msgs))
	}
}

func TestMailbox_MultipleMessages(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	mb := NewMailbox(store)

	senderPub, senderPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recipientPub, recipientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	senderDID := "did:mskt:sender"
	recipientDID := "did:mskt:recipient"

	SetKeyResolver(func(did string) (ed25519.PublicKey, error) {
		return senderPub, nil
	})

	for i := 0; i < 3; i++ {
		err := mb.Send(senderDID, recipientDID, []byte("message"), senderPriv, recipientPub)
		if err != nil {
			t.Fatalf("Send failed for message %d: %v", i, err)
		}
	}

	msgs, err := mb.Fetch(recipientDID, recipientPriv)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(msgs) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(msgs))
	}
}

func TestECDHRoundtrip(t *testing.T) {
	senderPub, senderPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recipientPub, recipientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	s1 := deriveSharedSecret(senderPriv, recipientPub)
	s2 := deriveSharedSecret(recipientPriv, senderPub)
	if len(s1) != 32 {
		t.Fatalf("shared secret 1 wrong length: %d", len(s1))
	}
	if hex.EncodeToString(s1) != hex.EncodeToString(s2) {
		t.Fatalf("ECDH mismatch: s1=%x s2=%x", s1, s2)
	}

	if hex.EncodeToString(s1) == hex.EncodeToString(senderPub)+hex.EncodeToString(recipientPub) {
		t.Fatal("shared secret is just concatenation - wrong")
	}
	t.Logf("ECDH OK: %x", s1)
}

func TestMailbox_WrongKeyDecryptFails(t *testing.T) {
	store := content.NewMemoryBlockStore(storage.NewQuotaManager())
	mb := NewMailbox(store)

	senderPub, senderPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recipientPub, recipientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, wrongPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	SetKeyResolver(func(did string) (ed25519.PublicKey, error) {
		return senderPub, nil
	})

	err = mb.Send("did:mskt:sender", "did:mskt:recipient", []byte("secret"), senderPriv, recipientPub)
	if err != nil {
		t.Fatal(err)
	}

	msgs, err := mb.Fetch("did:mskt:recipient", recipientPriv)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	_, err = mb.DecryptMessage(msgs[0], wrongPriv)
	if err == nil {
		t.Error("Expected decrypt error with wrong key, got nil")
	}
}
