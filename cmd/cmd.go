package cmd

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

// CmdCloser wraps the execution of a command with read, write, closer.
type CmdCloser struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	logger    *zap.Logger

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// NewCmdCloser creates a new CmdCloser.
// Starts a goroutine which monitors the process's status and closes the context when the process stops.
func NewCmdCloser(parentCtx context.Context, logger *zap.Logger, cmdStr string, cmdArgs []string) (*CmdCloser, error) {
	// Setup context
	ctx, cancelCtx := context.WithCancel(parentCtx)

	// Start process
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

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancelCtx()
		return nil, fmt.Errorf("failed to make stderr pipe: %s", err)
	}

	if err = cmd.Start(); err != nil {
		cancelCtx()
		return nil, fmt.Errorf("failed to run lsp: %s", err)
	}

	// Start watchdog goroutine
	cmdCloser := &CmdCloser{
		ctx:       ctx,
		cancelCtx: cancelCtx,
		logger:    logger,

		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}

	go cmdCloser.watchdog()

	return cmdCloser, nil
}

// watchdog monitors the underlying process's status and closes the context when the process exits.
func (cmd CmdCloser) watchdog() {
	cmdWaitChan := make(chan struct{})
	go func() {
		cmd.cmd.Wait()
		cmdWaitChan <- struct{}{}
	}()

	select {
	case <-cmdWaitChan:
		cmd.cancelCtx()
	case <-cmd.Done():
		return
	}
}

func (cmd CmdCloser) StderrLogger() {
	ticker := time.NewTicker(time.Millisecond * 200)

	line := ""

	for {
		select {
		case <-ticker.C:
			stderrBuff := make([]byte, 2048)
			n, err := cmd.ReadStderr(stderrBuff)
			if err != nil {
				cmd.logger.Error("failed to read stderr (used for debug msgs usually)", zap.Error(err))
			}

			if n > 0 {
				decoded := string(stderrBuff)

				for _, char := range decoded {
					charStr := string(char)

					if charStr == "\u0000" {
						continue
					}

					if charStr == "\n" && len(line) > 0 {
						cmd.logger.Debug("output", zap.String("line", line))
						line = ""
					} else if charStr != "\n" {
						line += charStr
					}
				}
			}
		case <-cmd.Done():
			if len(line) > 0 {
				cmd.logger.Debug("output", zap.String("line", line), zap.Bool("flush", true))
			}
			cmd.logger.Debug("process done")
			return
		}
	}
}

// Pid returns the ID of the underlying command process.
func (cmd CmdCloser) Pid() int {
	return cmd.cmd.Process.Pid
}

// Close stop the command execution and closes all stdin, stdout, and stderr pipes.
func (cmd CmdCloser) Close() error {
	cmd.cancelCtx()

	if err := cmd.stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %s", err)
	}

	if err := cmd.stdout.Close(); err != nil {
		return fmt.Errorf("failed to close stdout: %s", err)
	}

	if err := cmd.stderr.Close(); err != nil {
		return fmt.Errorf("failed to close stderr: %s", err)
	}

	return nil
}

// Done returns a channel which receives exactly one message when the command's context is closed.
// Context will be closed when command stops executing.
func (cmd CmdCloser) Done() <-chan struct{} {
	return cmd.ctx.Done()
}

// Read the command stdout.
func (cmd CmdCloser) Read(p []byte) (n int, err error) {
	return cmd.stdout.Read(p)
}

// Write to the command stdin.
func (cmd CmdCloser) Write(p []byte) (n int, err error) {
	return cmd.stdin.Write(p)
}

// ReadStderr reads a message from stderr.
func (cmd CmdCloser) ReadStderr(p []byte) (n int, err error) {
	return cmd.stderr.Read(p)
}
