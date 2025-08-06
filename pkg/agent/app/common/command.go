package common

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"sync"
)

// CommandExecutor encapsulates command execution logic
type CommandExecutor struct {
	ctx    context.Context
	logger *zap.SugaredLogger
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(ctx context.Context, logger *zap.SugaredLogger) *CommandExecutor {
	return &CommandExecutor{
		ctx:    ctx,
		logger: logger,
	}
}

// ExecutePipedCommands executes two commands with pipe connection
func (e *CommandExecutor) ExecutePipedCommands(cmd1 *exec.Cmd, cmd2 *exec.Cmd, logPrefix string) error {
	pr, pw := io.Pipe()
	cmd1.Stdout = pw
	cmd2.Stdin = pr

	// Get stderr pipes
	stderr1, err := cmd1.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get %s stderr pipe: %v", cmd1.Args[0], err)
	}

	stderr2, err := cmd2.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get %s stderr pipe: %v", cmd2.Args[0], err)
	}

	// Start commands
	e.logger.Infof("starting %s command...", cmd1.Args[0])
	if err := cmd1.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %v", cmd1.Args[0], err)
	}

	e.logger.Infof("starting %s command...", cmd2.Args[0])
	if err := cmd2.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %v", cmd2.Args[0], err)
	}

	// Handle stderr logging in goroutines
	wg := sync.WaitGroup{}
	logDir := os.Getenv("LOG_MOUNT")
	wg.Add(3)

	go e.handleStderr(&wg, stderr1, fmt.Sprintf("%s/%s-%s.log", logDir, cmd1.Args[0], logPrefix))
	go e.handleStderr(&wg, stderr2, fmt.Sprintf("%s/%s-%s.log", logDir, cmd2.Args[0], logPrefix))
	
	go func() {
		defer pw.Close()
		defer wg.Done()
		cmd1.Wait()
	}()

	// Wait for second command
	e.logger.Infof("waiting for %s command to finish...", cmd2.Args[0])
	if err := cmd2.Wait(); err != nil {
		return fmt.Errorf("failed to execute %s: %v", cmd2.Args[0], err)
	}

	wg.Wait()
	return nil
}

// handleStderr handles stderr logging for commands
func (e *CommandExecutor) handleStderr(wg *sync.WaitGroup, stderr io.ReadCloser, logPath string) {
	defer wg.Done()
	
	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		e.logger.Errorf("failed to read stderr: %v", err)
		return
	}
	
	if err := os.WriteFile(logPath, stderrBytes, 0644); err != nil {
		e.logger.Errorf("failed to write stderr to file %s: %v", logPath, err)
	}
}

// ExecuteCommand executes a single command with stderr logging
func (e *CommandExecutor) ExecuteCommand(cmd *exec.Cmd, logPrefix string) error {
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %v", err)
	}

	e.logger.Infof("starting %s command...", cmd.Args[0])
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %v", cmd.Args[0], err)
	}

	// Handle stderr
	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		e.logger.Errorf("failed to read %s stderr: %v", cmd.Args[0], err)
	}

	logDir := os.Getenv("LOG_MOUNT")
	logPath := fmt.Sprintf("%s/%s-%s.log", logDir, cmd.Args[0], logPrefix)
	if err := os.WriteFile(logPath, stderrBytes, 0644); err != nil {
		e.logger.Errorf("failed to write %s stderr to file %s: %v", cmd.Args[0], logPath, err)
	}

	e.logger.Infof("waiting for %s to finish...", cmd.Args[0])
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to execute %s: %v, stderr: %s", cmd.Args[0], err, string(stderrBytes))
	}

	return nil
}