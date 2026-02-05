package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	runErr := fn()

	require.NoError(t, w.Close())
	os.Stdout = orig

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	return buf.String(), runErr
}

func TestRootCommandPrintsVersion(t *testing.T) {
	rootCmd.SetArgs([]string{"--version"})
	vers = false

	output, err := captureStdout(t, rootCmd.Execute)
	require.NoError(t, err)
	require.Contains(t, output, "Version")

	vers = false
	rootCmd.SetArgs(nil)
}

func TestRootCommandRequiresFlag(t *testing.T) {
	vers = false
	rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	require.EqualError(t, err, "no flags find")
}
