package main

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"

	"context"
	"fmt"
	"io"
	"os/exec"
)

type CmdCloser struct {
	cancelCtx context.CancelFunc
	logger    *zap.Logger

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func NewCmdCloser(parentCtx context.Context, logger *zap.Logger, cmdStr string) (*CmdCloser, error) {
	ctx, cancelCtx := context.WithCancel(parentCtx)

	cmd := exec.CommandContext(ctx, cmdStr)
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

	/* cmd.std
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancelCtx()
		return nil, fmt.Errorf("failed to make stderr pipe: %s", err)
	}

	go func() {
		stderr.Read()
	}()
	*/
	return &CmdCloser{
		cancelCtx: cancelCtx,
		logger:    logger,
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
	}, nil
}

func (cmd CmdCloser) Close() error {
	cmd.cancelCtx()
	cmd.logger.Debug("closed")
	return nil
}

func (cmd CmdCloser) Read(p []byte) (n int, err error) {
	cmd.logger.Debug("read called")
	n, err = cmd.stdout.Read(p)
	cmd.logger.Debug("read", zap.ByteString("p", p), zap.Int("n", n), zap.Error(err))
	if err != nil {
		return n, err
	}
	return n, err
}

func (cmd CmdCloser) Write(p []byte) (n int, err error) {
	cmd.logger.Debug("write", zap.ByteString("p", p))
	return cmd.stdin.Write(p)
}

func main() {
	ctx := context.Background()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to make logger: %s", err))
	}
	defer logger.Sync()

	proc, err := NewCmdCloser(ctx, logger, "clangd")
	if err != nil {
		logger.Fatal("failed to run C lsp", zap.Error(err))
	}

	logger.Info("running lsp")

	stream := jsonrpc2.NewStream(proc)
	conn := jsonrpc2.NewConn(stream)

	client := protocol.ServerDispatcher(conn, logger)

	logger.Info("listing folders")
	folders, err := client.Initialize(ctx, &protocol.InitializeParams{})
	if err != nil {
		logger.Fatal("failed to list folders", zap.Error(err))
	}

	logger.Info("folders", zap.Any("folders", folders))
}
