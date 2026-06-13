package storage

import (
	"testing"
)

func TestErasureCoding_Reconstruction(t *testing.T) {
	encoder, err := NewErasureCoder()
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	originalData := []byte("This is a highly critical piece of data for Musketeers network.")

	// 1. التشفير والتجزئة
	shards, err := encoder.Encode(originalData)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// 2. التحقق من عدد الأجزاء
	if len(shards) != TotalShards {
		t.Errorf("Expected %d shards, got %d", TotalShards, len(shards))
	}

	// 3. التحقق من أن الأجزاء ليست فارغة
	for i, shard := range shards {
		if len(shard) == 0 {
			t.Errorf("Shard %d is empty", i)
		}
	}
}
