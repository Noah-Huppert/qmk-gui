package main

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"

	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
)

type CmdCloser struct {
	ctx context.Context

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func NewCmdCloser(ctx context.Context, cmdStr string) (*CmdCloser, error) {
	cmd := exec.CommandContext(ctx, cmdStr)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to make stdin pipe: %s", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to make stdout pipe: %s", err)
	}

	return &CmdCloser{
		ctx:    ctx,
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}

func (cmd CmdCloser) Close() error {
	if err := cmd.Close(); err != nil {
		return fmt.Errorf("error closing command context: %s", err)
	}

	return nil
}

func (cmd CmdCloser) Read(p []byte) (n int, err error) {
	return cmd.stdout.Read(p)
}

func (cmd CmdCloser) Write(p []byte) (n int, err error) {
	return cmd.stdin.Write(p)
}

func main() {
	ctx := context.Background()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to make logger: %s", err)
	}
	defer logger.Sync()

	proc, err := NewCmdCloser(ctx, "clangd")
	if err != nil {
		log.Fatalf("failed to run C lsp: %s", err)
	}

	stream := jsonrpc2.NewStream(proc)
	conn := jsonrpc2.NewConn(stream)

	client := protocol.ClientDispatcher(conn, logger)

	logger.Info("hi client", zap.Any("client", client))
}
