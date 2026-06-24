package node

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const projectPrefix = "/mskt/"
const oldPrefix = "/nr/"

func TestDHT_PrefixesCorrect(t *testing.T) {
	goFiles, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Failed to list go files: %v", err)
	}

	for _, f := range goFiles {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", f, err)
		}

		contentStr := string(content)

		if strings.Contains(contentStr, `"`+oldPrefix) {
			t.Errorf("%s still contains old %s prefix", f, oldPrefix)
		}
	}
}
