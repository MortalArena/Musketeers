package node

import (
	"os"
	"strings"
	"testing"
)

func TestDHT_PrefixesCorrect(t *testing.T) {
	// قراءة محتوى node.go
	content, err := os.ReadFile("node.go")
	if err != nil {
		t.Fatalf("Failed to read node.go: %v", err)
	}

	contentStr := string(content)

	// التأكد من عدم وجود /nr/
	if strings.Contains(contentStr, `"/nr/`) {
		t.Errorf("node.go still contains old /nr/ prefix")
	}

	// التأكد من وجود /mskt/
	requiredPrefixes := []string{
		`"/mskt/identity/"`,
		`"/mskt/revoke/"`,
	}

	for _, prefix := range requiredPrefixes {
		if !strings.Contains(contentStr, prefix) {
			t.Errorf("node.go missing required prefix: %s", prefix)
		}
	}
}
