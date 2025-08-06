package version

import (
	"fmt"
)

const (
	// ServiceName service name
	ServiceName = "unit-agent"
)

var (
	GIT_TAG    string = "v2.0.1"
	GIT_COMMIT string
	GIT_BRANCH string
	BUILD_TIME string
	GO_VERSION string
)

// FullVersion show the version info
func FullVersion() string {
	version := fmt.Sprintf("Version   : %s\nBuild Time: %s\nGit Branch: %s\nGit Commit: %s\nGo Version: %s\n", GIT_TAG, BUILD_TIME, GIT_BRANCH, GIT_COMMIT, GO_VERSION)
	return version
}

// Short short version
func Short() string {
	commit := ""
	if len(GIT_COMMIT) > 8 {
		commit = GIT_COMMIT[:8]
	}
	return fmt.Sprintf("%s[%s %s]", GIT_TAG, BUILD_TIME, commit)
}
