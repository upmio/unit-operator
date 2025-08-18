package version

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFullVersion(t *testing.T) {
	// Define test cases
	tests := []struct {
		name           string
		gitTag         string
		gitCommit      string
		gitBranch      string
		buildTime      string
		goVersion      string
		expectedOutput string
	}{
		{
			name:           "all fields set",
			gitTag:         "v1.0.1",
			gitCommit:      "abcd1234",
			gitBranch:      "main",
			buildTime:      "2024-08-26T12:00:00Z",
			goVersion:      "go1.20",
			expectedOutput: "Version   : v1.0.1\nBuild Time: 2024-08-26T12:00:00Z\nGit Branch: main\nGit Commit: abcd1234\nGo Version: go1.20\n",
		},
		{
			name:           "empty fields",
			gitTag:         "",
			gitCommit:      "",
			gitBranch:      "",
			buildTime:      "",
			goVersion:      "",
			expectedOutput: "Version   : \nBuild Time: \nGit Branch: \nGit Commit: \nGo Version: \n",
		},
		{
			name:           "partial fields set",
			gitTag:         "v1.0.1",
			gitCommit:      "",
			gitBranch:      "feature-branch",
			buildTime:      "2024-08-26T12:00:00Z",
			goVersion:      "go1.19",
			expectedOutput: "Version   : v1.0.1\nBuild Time: 2024-08-26T12:00:00Z\nGit Branch: feature-branch\nGit Commit: \nGo Version: go1.19\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global variables
			GIT_TAG = tt.gitTag
			GIT_COMMIT = tt.gitCommit
			GIT_BRANCH = tt.gitBranch
			BUILD_TIME = tt.buildTime
			GO_VERSION = tt.goVersion

			// Get the output from FullVersion function
			actualOutput := FullVersion()

			// Assert the output
			assert.Equal(t, tt.expectedOutput, actualOutput)
		})
	}
}
