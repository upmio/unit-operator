package common

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
)

// CommandExecutor encapsulates command execution logic
type CommandExecutor struct {
	logger *zap.SugaredLogger
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(logger *zap.SugaredLogger) *CommandExecutor {
	return &CommandExecutor{
		logger: logger,
	}
}

// ExecutePipedCommands executes two commands with pipe connection
func (e *CommandExecutor) ExecutePipedCommands(cmd1 *exec.Cmd, cmd2 *exec.Cmd, logPrefix string) error {
	if err := e.prepareCommand(cmd1); err != nil {
		return err
	}

	logFile1, err := e.openLogFile(cmd1.Args[0], logPrefix)
	if err != nil {
		return err
	}
	defer func() { _ = logFile1.Close() }()

	if err := e.prepareCommand(cmd2); err != nil {
		return err
	}

	logFile2, err := e.openLogFile(cmd2.Args[0], logPrefix)
	if err != nil {
		return err
	}
	defer func() { _ = logFile2.Close() }()

	pr, pw := io.Pipe()
	cmd1.Stdout = pw
	cmd2.Stdin = pr

	stderr1, _ := cmd1.StderrPipe()
	stderr2, _ := cmd2.StderrPipe()

	e.logger.Infof("starting command (pip command): %s", strings.Join(cmd1.Args, " "))
	if err := cmd1.Start(); err != nil {
		return err
	}

	e.logger.Infof("starting command (pip command):  %s", strings.Join(cmd2.Args, " "))
	if err := cmd2.Start(); err != nil {
		return err
	}

	go io.Copy(logFile1, stderr1)
	go io.Copy(logFile2, stderr2)

	errCh := make(chan error, 2)

	go func() {
		errCh <- cmd1.Wait()
		_ = pw.Close()
	}()

	go func() {
		errCh <- cmd2.Wait()
		_ = pr.Close()
	}()

	err1 := <-errCh
	err2 := <-errCh

	if err1 != nil {
		return fmt.Errorf("command %s failed (see %s)", cmd1.Args[0], logFile1.Name())
	}
	if err2 != nil {
		return fmt.Errorf("command %s failed (see %s)", cmd2.Args[0], logFile2.Name())
	}

	return nil
}

// ExecuteCommand executes a single command with stderr logging
func (e *CommandExecutor) ExecuteCommand(cmd *exec.Cmd, logPrefix string) error {
	if err := e.prepareCommand(cmd); err != nil {
		return err
	}

	logFile, err := e.openLogFile(cmd.Args[0], logPrefix)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	cmd.Stderr = logFile
	cmd.Stdout = logFile

	e.logger.Infof("starting command: %s", strings.Join(cmd.Args, " "))

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command failed: %w (see %s)", err, logFile.Name())
	}

	return nil
}

func (e *CommandExecutor) ExecuteCommandStreamFromS3(ctx context.Context, cmd *exec.Cmd, factory ObjectStorageFactory, bucket, object, logPrefix string) error {
	if err := e.prepareCommand(cmd); err != nil {
		return err
	}

	logFile, err := e.openLogFile(cmd.Args[0], logPrefix)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	// 获取 S3 对象（reader）
	objReader, err := factory.GetObject(ctx, bucket, object)
	if err != nil {
		return fmt.Errorf("get object from s3 failed: %w", err)
	}
	defer func() { _ = objReader.Close() }()

	pr, pw := io.Pipe()
	cmd.Stdin = pr
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	e.logger.Infof("starting command (streaming from s3): %s", strings.Join(cmd.Args, " "))

	if err := cmd.Start(); err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return err
	}

	cmdErrCh := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		if err != nil {
			_ = pr.CloseWithError(err)
		} else {
			_ = pr.Close()
		}
		cmdErrCh <- err
	}()

	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(pw, objReader)
		_ = pw.Close()
		copyErrCh <- err
	}()

	copyErr := <-copyErrCh
	cmdErr := <-cmdErrCh

	if copyErr != nil {
		return fmt.Errorf("streaming from s3 failed: %w", copyErr)
	}

	if cmdErr != nil {
		return fmt.Errorf("command failed: %w (see %s)", cmdErr, logFile.Name())
	}

	return nil
}

func (e *CommandExecutor) ExecuteCommandStreamToS3(ctx context.Context, cmd *exec.Cmd, factory ObjectStorageFactory, bucket, object, logPrefix string) error {
	if err := e.prepareCommand(cmd); err != nil {
		return err
	}

	logFile, err := e.openLogFile(cmd.Args[0], logPrefix)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = logFile

	e.logger.Infof("starting command (streaming to s3): %s", strings.Join(cmd.Args, " "))

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return err
	}

	// command execution goroutine
	cmdErrCh := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		if err != nil {
			_ = pw.CloseWithError(err)
		} else {
			_ = pw.Close()
		}
		cmdErrCh <- err
	}()

	// upload blocks until EOF or error
	uploadErr := factory.PutObject(ctx, bucket, object, pr)
	_ = pr.Close()

	cmdErr := <-cmdErrCh

	if cmdErr != nil {
		return fmt.Errorf("command failed: %w (see %s)", cmdErr, logFile.Name())
	}

	if uploadErr != nil {
		return fmt.Errorf("upload to s3 failed: %w", uploadErr)
	}

	return nil
}

func (e *CommandExecutor) prepareCommand(cmd *exec.Cmd) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("empty command")
	}

	if _, err := exec.LookPath(cmd.Args[0]); err != nil {
		return fmt.Errorf("%s not found in PATH: %w", cmd.Args[0], err)
	}
	return nil
}

func (e *CommandExecutor) openLogFile(cmdName, logPrefix string) (*os.File, error) {
	logDir, err := util.IsEnvVarSet(vars.LogMountEnvKey)
	if err != nil {
		return nil, err
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", cmdName, logPrefix))
	return os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}
