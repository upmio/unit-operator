package version

import (
	"strings"
	"testing"
)

func TestFullVersionIncludesFields(t *testing.T) {
	output := FullVersion()
	for _, segment := range []string{"Version", "Build Time", "Git Branch", "Git Commit", "Go Version"} {
		if !strings.Contains(output, segment) {
			t.Fatalf("expected %s in full version output", segment)
		}
	}
}

func TestShortVersionUsesCommitPrefix(t *testing.T) {
	GIT_TAG = "v1.0.0"
	GIT_COMMIT = "1234567890abcdef"
	BUILD_TIME = "2024-01-01"

	short := Short()
	if !strings.Contains(short, "12345678") {
		t.Fatalf("expected commit prefix in short version: %s", short)
	}
	if !strings.Contains(short, GIT_TAG) {
		t.Fatalf("expected tag in short version: %s", short)
	}
}
