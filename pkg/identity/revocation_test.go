package identity

import (
	"strings"
	"testing"
)

func TestRevocationRecord_DHTKey(t *testing.T) {
	rec := &RevocationRecord{DID: "did:mskt:test123"}
	expected := "/mskt/revoke/did:mskt:test123"

	if got := rec.DHTKey(); got != expected {
		t.Errorf("DHTKey() = %v, want %v", got, expected)
	}

	// التأكد من عدم وجود /nr/
	if strings.Contains(rec.DHTKey(), "/nr/") {
		t.Errorf("DHTKey() contains old /nr/ prefix: %v", rec.DHTKey())
	}
}
