package cmd

import (
	"errors"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/version"
	"os"

	"github.com/spf13/cobra"
)

var (
	vers bool
)

// RootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "unit-agent is used to wrapper configmap or secret object in kubernetes.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if vers {
			fmt.Println(version.FullVersion())
			return nil
		}
		return errors.New("No flags find")
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		//fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&vers, "version", "v", false, "Print the version information")
	rootCmd.AddCommand(daemonCmd)
}
