package main

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"

	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	return nil
}

func (cmd CmdCloser) Read(p []byte) (n int, err error) {
	return cmd.stdout.Read(p)
}

func (cmd CmdCloser) Write(p []byte) (n int, err error) {
	return cmd.stdin.Write(p)
}

type ClientHandler struct {
	logger *zap.Logger
}

func (h ClientHandler) Handle(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	h.logger.Debug("received response over connection", zap.Any("req", req))
	return nil
}

func main() {
	// Setup context and logger
	ctx := context.Background()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to make logger: %s", err))
	}
	defer logger.Sync()

	// Start LSP server
	proc, err := NewCmdCloser(ctx, logger, "clangd")
	if err != nil {
		logger.Fatal("failed to run C LSP", zap.Error(err))
	}

	logger.Info("running lsp")

	stream := jsonrpc2.NewStream(proc)
	conn := jsonrpc2.NewConn(stream)

	go func() {
		handler := ClientHandler{
			logger: logger,
		}
		conn.Go(ctx, handler.Handle)
	}()

	client := protocol.ServerDispatcher(conn, logger)

	// Initialize LSP
	logger.Info("initializing C LSP")

	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatal("failed to get working directory", zap.Error(err))
	}

	qmkFirmwareDir := filepath.Join(cwd, "../qmk_firmware")

	_, err = client.Initialize(ctx, &protocol.InitializeParams{
		ClientInfo: &protocol.ClientInfo{
			Name:    "qmk-gui",
			Version: "pre-alpha",
		},
		Locale: "en",
		Capabilities: protocol.ClientCapabilities{
			Workspace: &protocol.WorkspaceClientCapabilities{
				WorkspaceFolders: true,
			},
		},
		WorkspaceFolders: []protocol.WorkspaceFolder{
			{
				Name: "qmk_firmware",
				URI:  fmt.Sprintf("file://%s", qmkFirmwareDir),
			},
		},
	})
	if err != nil {
		logger.Fatal("failed to initialize C LSP", zap.Error(err))
	}

	logger.Info("initialized C LSP")

	keymapCFilePath := filepath.Join(qmkFirmwareDir, "keyboards/moonlander/keymaps/default/keymap.c")
	keymapCFileBytes, err := os.ReadFile(keymapCFilePath)
	if err != nil {
		logger.Fatal("failed to read keymap.c file", zap.Error(err))
	}
	keymapCFile := bytes.NewBuffer(keymapCFileBytes).String()

	logger.Debug("keymap.c", zap.String("contents", keymapCFile))

	client.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri.File(keymapCFilePath),
			LanguageID: "c",
			Version:    0,
			Text:       keymapCFile,
		},
	})

	symbols, err := client.Symbols(ctx, &protocol.WorkspaceSymbolParams{})
	if err != nil {
		logger.Fatal("failed to list symbols", zap.Error(err))
	}
	logger.Debug("retrieved symbols", zap.Any("symbols", symbols))
}
