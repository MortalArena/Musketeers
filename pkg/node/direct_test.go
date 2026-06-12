package node

import (
	"crypto/ed25519"
	"testing"
)

func TestDirectMessageEncryptionDecryption(t *testing.T) {
	pubSender, privSender, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	pubRecipient, privRecipient, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	from := "did:ia:sender"
	to := "did:ia:recipient"
	content := []byte("hello, this is a secure direct message!")

	msg, err := EncryptDirectMessage(from, to, content, privSender, pubRecipient)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	decrypted, err := DecryptDirectMessage(msg, privRecipient, pubSender)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if string(decrypted) != string(content) {
		t.Errorf("decrypted content mismatch: expected %q, got %q", string(content), string(decrypted))
	}
}

func TestChunkyMessageRoundtrip(t *testing.T) {
	pubSender, privSender, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	pubRecipient, privRecipient, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	from := "did:ia:sender"
	to := "did:ia:recipient"
	// Generate large content that exceeds the MaxChunkSize
	content := make([]byte, 300*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	chunks, err := ChunkMessage(from, to, content, privSender, pubRecipient)
	if err != nil {
		t.Fatalf("failed to chunk and encrypt: %v", err)
	}

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	assembler := NewChunkAssembler()
	var finalData []byte
	var complete bool

	for _, chunk := range chunks {
		decryptedChunk, err := DecryptDirectMessage(chunk, privRecipient, pubSender)
		if err != nil {
			t.Fatalf("failed to decrypt chunk: %v", err)
		}

		finalData, complete, _ = assembler.Add(chunk, decryptedChunk)
	}

	if !complete {
		t.Fatal("expected message assembly to be complete")
	}

	if len(finalData) != len(content) {
		t.Fatalf("assembled size mismatch: expected %d, got %d", len(content), len(finalData))
	}

	for i := range content {
		if finalData[i] != content[i] {
			t.Fatalf("byte at index %d mismatch", i)
		}
	}
}
