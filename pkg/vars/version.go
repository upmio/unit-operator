package vars

import (
	"fmt"

	"github.com/fatih/color"
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

func PrintVersion() {

	tempVer := "dev"
	if VERSION != "" {
		tempVer = VERSION
	}

	bold := color.New(color.FgCyan, color.Bold).SprintFunc()

	fmt.Printf("\n%s\n", bold("=== Unit Operator Version Info ==="))
	fmt.Printf("%-12s: %s\n", "Version", tempVer)
	fmt.Printf("%-12s: %s\n", "Build Time", BUILDTIME)
	fmt.Printf("%-12s: %s\n", "Git Branch", GITBRANCH)
	fmt.Printf("%-12s: %s\n", "Git Commit", GITCOMMIT)
	fmt.Printf("%-12s: %s\n", "Go Version", GOVERSION)
	fmt.Printf("%s\n", bold("====================================\n"))
}
