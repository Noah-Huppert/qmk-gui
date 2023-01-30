package cmd

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"go.uber.org/zap"
)

// Wraps the execution of a command in a read, write, closer.
type CmdCloser struct {
	cancelCtx context.CancelFunc
	logger    *zap.Logger

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

// Creates a new CmdCloser.
func NewCmdCloser(parentCtx context.Context, logger *zap.Logger, cmdStr string, cmdArgs []string) (*CmdCloser, error) {
	ctx, cancelCtx := context.WithCancel(parentCtx)

	cmd := exec.CommandContext(ctx, cmdStr, cmdArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancelCtx()
		return nil, fmt.Errorf("failed to make stdin pipe: %s", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancelCtx()
		return nil, fmt.Errorf("failed to make stdout pipe: %s", err)
	}

	if err = cmd.Start(); err != nil {
		cancelCtx()
		return nil, fmt.Errorf("failed to run lsp: %s", err)
	}

	return &CmdCloser{
		cancelCtx: cancelCtx,
		logger:    logger,
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
	}, nil
}

func (cmd CmdCloser) Pid() int {
	return cmd.cmd.Process.Pid
}

// Stop the command execution.
func (cmd CmdCloser) Close() error {
	cmd.cancelCtx()
	return nil
}

// Read the command stdout.
func (cmd CmdCloser) Read(p []byte) (n int, err error) {
	return cmd.stdout.Read(p)
}

// Write to the command stdin.
func (cmd CmdCloser) Write(p []byte) (n int, err error) {
	return cmd.stdin.Write(p)
}
