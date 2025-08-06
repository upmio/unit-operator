package vars

import (
	"fmt"
)

var (
	// GITCOMMIT will be overwritten automatically by the build system with short commit hash
	GITCOMMIT = "HEAD"
	// BUILDTIME will be overwritten automatically by the build system
	BUILDTIME = "<unknown>"
	// VERSION will be overwritten automatically by the build system if tag exists at current commit
	VERSION = ""
	// GITBRANCH will be overwritten automatically by the build system with the current branch
	GITBRANCH string
	// GOVERSION will be overwritten automatically by the build system with the go version in build image
	GOVERSION string
)

const (
	//Version           = "1.1.0"
	VersionPrerelease = ""
)

type VersionInfo struct {
	Version   string `json:"version"`
	CommitId  string `json:"commit_id"`
	BuildTime string `json:"build_time"`
	GitBranch string `json:"git_branch"`
	GoVersion string `json:"go_version"`
}

func GetVersion() string {
	tempVer := "dev"
	if VERSION != "" {
		tempVer = VERSION
	} else {
		tempVer = "dev"
	}

	version := fmt.Sprintf(
		"Version: %s\n"+
			"Build Time: %s\n"+
			"Git Branch: %s\n"+
			"Git Commit: %s\n"+
			"Go Version: %s\n",
		tempVer,
		BUILDTIME,
		GITBRANCH,
		GITCOMMIT,
		GOVERSION)

	return version
}
