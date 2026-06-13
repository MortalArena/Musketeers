package crypto

import (
	"bytes"
	"testing"
)

func TestShamirSecretSharing(t *testing.T) {
	originalKey := []byte("super-secret-master-key-12345")

	// 1. التقسيم
	shares, err := SplitMasterKey(originalKey)
	if err != nil {
		t.Fatalf("Failed to split key: %v", err)
	}
	if len(shares) != TotalShares {
		t.Errorf("Expected %d shares, got %d", TotalShares, len(shares))
	}

	// 2. محاكاة استخدام جزأين فقط (تجاهل الجزء الثالث)
	recoveryShares := [][]byte{shares[0], shares[2]}

	// 3. إعادة البناء
	reconstructedKey, err := ReconstructMasterKey(recoveryShares)
	if err != nil {
		t.Fatalf("Failed to reconstruct key: %v", err)
	}

	// 4. التحقق من التطابق التام
	if !bytes.Equal(originalKey, reconstructedKey) {
		t.Errorf("Reconstructed key does not match original")
	}

	// 5. اختبار الفشل: محاولة البناء بجزء واحد فقط
	_, err = ReconstructMasterKey([][]byte{shares[0]})
	if err == nil {
		t.Errorf("Expected error when providing insufficient shares, but got none")
	}
}
