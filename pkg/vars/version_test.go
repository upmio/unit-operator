package vars

import (
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

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		commit   string
		build    string
		version  string
		branch   string
		gover    string
		expected string
	}{
		{
			name:     "Default values",
			commit:   "HEAD",
			build:    "<unknown>",
			version:  "",
			branch:   "",
			gover:    "",
			expected: "Version: dev\nBuild Time: <unknown>\nGit Branch: \nGit Commit: HEAD\nGo Version: \n",
		},
		{
			name:     "Specific values",
			commit:   "abc123",
			build:    "2024-01-01T00:00:00Z",
			version:  "v1.0.0",
			branch:   "main",
			gover:    "go1.18",
			expected: "Version: v1.0.0\nBuild Time: 2024-01-01T00:00:00Z\nGit Branch: main\nGit Commit: abc123\nGo Version: go1.18\n",
		},
		{
			name:     "VERSION is empty",
			commit:   "abc123",
			build:    "2024-01-01T00:00:00Z",
			version:  "",
			branch:   "main",
			gover:    "go1.18",
			expected: "Version: dev\nBuild Time: 2024-01-01T00:00:00Z\nGit Branch: main\nGit Commit: abc123\nGo Version: go1.18\n",
		},
		{
			name:     "GITBRANCH is empty",
			commit:   "abc123",
			build:    "2024-01-01T00:00:00Z",
			version:  "",
			branch:   "",
			gover:    "go1.18",
			expected: "Version: dev\nBuild Time: 2024-01-01T00:00:00Z\nGit Branch: \nGit Commit: abc123\nGo Version: go1.18\n",
		},
		{
			name:     "GOVERSION is empty",
			commit:   "abc123",
			build:    "2024-01-01T00:00:00Z",
			version:  "",
			branch:   "main",
			gover:    "",
			expected: "Version: dev\nBuild Time: 2024-01-01T00:00:00Z\nGit Branch: main\nGit Commit: abc123\nGo Version: \n",
		},
		{
			name:     "All empty",
			commit:   "",
			build:    "",
			version:  "",
			branch:   "",
			gover:    "",
			expected: "Version: dev\nBuild Time: \nGit Branch: \nGit Commit: \nGo Version: \n",
		},
		{
			name:     "Only version set",
			commit:   "",
			build:    "",
			version:  "v1.2.3",
			branch:   "",
			gover:    "",
			expected: "Version: v1.2.3\nBuild Time: \nGit Branch: \nGit Commit: \nGo Version: \n",
		},
		{
			name:     "Only commit set",
			commit:   "def456",
			build:    "",
			version:  "",
			branch:   "",
			gover:    "",
			expected: "Version: dev\nBuild Time: \nGit Branch: \nGit Commit: def456\nGo Version: \n",
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

			assert.Equal(t, tt.expected, GetVersion())
		})
	}
}
