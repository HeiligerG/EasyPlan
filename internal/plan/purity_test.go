package plan

import (
	"os"
	"strings"
	"testing"
)

// TestPackage_does_not_read_the_current_time protects the rule of the concept
// (section 7): the calculation functions are pure, the caller passes "today"
// explicitly. A call of time.Now inside the package would make every result
// depend on the moment of the call.
func TestPackage_does_not_read_the_current_time(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("cannot read the package directory: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("cannot read %s: %v", name, err)
		}
		if strings.Contains(string(content), "time.Now(") {
			t.Errorf("%s calls time.Now(), the package must stay pure", name)
		}
	}
}
