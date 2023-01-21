package main

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"

	"context"
	"fmt"
	"os"
	"os/exec"
)

type CmdCloser struct {
	ctx context.Context

	reader *os.File
	writer *os.File
	cmd    *exec.Cmd
}

func NewCmdCloser(ctx context.Context, cmd string) (*CmdCloser, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create LSP io pipe: %s", err)
	}

	res := exec.CommandContext(ctx, cmd)
	res.Stdout = writer
	res.Stdin = reader

	return &CmdCloser{
		ctx:    ctx,
		reader: reader,
		writer: writer,
		cmd:    res,
	}, nil
}

func (cmd CmdCloser) Close() error {
	if err := cmd.Close(); err != nil {
		return fmt.Errorf("error closing command context: %s", err)
	}

	return nil
}

func (cmd CmdCloser) Read(p []byte) (n int, err error) {
	return cmd.reader.Read(p)
}

func (cmd CmdCloser) Write(p []byte) (n int, err error) {
	return cmd.writer.Write(p)
}

func main() {
	ctx := context.Background()

	os.Pipe

	proc := exec.CommandContext(ctx, "clangd")

	stream := jsonrpc2.NewStream(proc)
	conn := jsonrpc2.NewConn(stream)

	logger := zap.L()
	client := protocol.ClientDispatcher(conn, logger)

	logger.Info("hi client: %s", client)
}
