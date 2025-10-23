package vars

import (
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// helper function to reset global variables to default values
func resetGlobals() {
	GITCOMMIT = "HEAD"
	BUILDTIME = "<unknown>"
	VERSION = ""
	GITBRANCH = ""
	GOVERSION = ""
}

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	outBytes, _ := io.ReadAll(r)
	return string(outBytes)
}

func stripANSIEscape(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name    string
		commit  string
		build   string
		version string
		branch  string
		gover   string
		expect  map[string]string
	}{
		{
			name:    "Default values",
			commit:  "HEAD",
			build:   "<unknown>",
			version: "",
			branch:  "",
			gover:   "",
			expect: map[string]string{
				"Version":    "dev",
				"Build Time": "<unknown>",
				"Git Branch": "",
				"Git Commit": "HEAD",
				"Go Version": "",
			},
		},
		{
			name:    "Specific values",
			commit:  "abc123",
			build:   "2024-01-01T00:00:00Z",
			version: "v1.0.0",
			branch:  "main",
			gover:   "go1.18",
			expect: map[string]string{
				"Version":    "v1.0.0",
				"Build Time": "2024-01-01T00:00:00Z",
				"Git Branch": "main",
				"Git Commit": "abc123",
				"Go Version": "go1.18",
			},
		},
		{
			name:    "VERSION is empty",
			commit:  "abc123",
			build:   "2024-01-01T00:00:00Z",
			version: "",
			branch:  "main",
			gover:   "go1.18",
			expect: map[string]string{
				"Version":    "dev",
				"Build Time": "2024-01-01T00:00:00Z",
				"Git Branch": "main",
				"Git Commit": "abc123",
				"Go Version": "go1.18",
			},
		},
		{
			name:    "All empty",
			commit:  "",
			build:   "",
			version: "",
			branch:  "",
			gover:   "",
			expect: map[string]string{
				"Version":    "dev",
				"Build Time": "",
				"Git Branch": "",
				"Git Commit": "",
				"Go Version": "",
			},
		},
		{
			name:    "Only version set",
			commit:  "",
			build:   "",
			version: "v1.2.3",
			branch:  "",
			gover:   "",
			expect: map[string]string{
				"Version":    "v1.2.3",
				"Build Time": "",
				"Git Branch": "",
				"Git Commit": "",
				"Go Version": "",
			},
		},
		{
			name:    "Only commit set",
			commit:  "def456",
			build:   "",
			version: "",
			branch:  "",
			gover:   "",
			expect: map[string]string{
				"Version":    "dev",
				"Build Time": "",
				"Git Branch": "",
				"Git Commit": "def456",
				"Go Version": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobals()
			GITCOMMIT = tt.commit
			BUILDTIME = tt.build
			VERSION = tt.version
			GITBRANCH = tt.branch
			GOVERSION = tt.gover

			out := captureOutput(PrintVersion)
			clean := stripANSIEscape(out)
			clean = strings.ReplaceAll(clean, "\r\n", "\n")

			found := map[string]string{}
			for _, line := range strings.Split(clean, "\n") {
				if !strings.Contains(line, ":") {
					continue
				}
				parts := strings.SplitN(line, ":", 2)
				key := strings.TrimSpace(parts[0])
				val := ""
				if len(parts) > 1 {
					val = strings.TrimSpace(parts[1])
				}
				found[key] = val
			}

			for k, v := range tt.expect {
				got, ok := found[k]
				assert.True(t, ok, "expected key %s present in output", k)
				assert.Equal(t, v, got, "value mismatch for key %s", k)
			}
		})
	}
}
