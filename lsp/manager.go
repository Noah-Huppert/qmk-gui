package lsp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Noah-Huppert/qmk-gui/clangdlsp"
	"github.com/Noah-Huppert/qmk-gui/cmd"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// LSPManager wraps interactions with the language server.
type LSPManager struct {
	// ctx is the context used to manage the lifecycle of all operations.
	ctx context.Context

	// logger outputs debug and error information
	logger *zap.Logger

	// stream used to communicate with the LSP server.
	stream jsonrpc2.Stream

	// conn is the JSON RPC connection which provides a JSON RPC transport.
	conn jsonrpc2.Conn

	// server is a client for the LSP server which sends and receives LSP information
	server clangdlsp.ClangdServer

	// lspNotifications is used to communicate when different notifications are received from the LSP server
	lspNotifications LSPNotifications

	// docColl is used to manage the lifecycle of files opened by the LSP server
	docColl LSPDocumentCollection
}

// HandleLSPMsg runs when the JSON RPC connection communicating with the LSP server receives a message.
func (manager LSPManager) HandleLSPMsg(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	// Handle known notifications
	if req.Method() == protocol.MethodProgress {
		// Progress notification
		params := protocol.ProgressParams{}
		if err := json.Unmarshal(req.Params(), &params); err != nil {
			return fmt.Errorf("failed to unmarshall progress notification params: %s", err)
		}

		// Handle known progress tokens
		if params.Token.String() == clangdlsp.ProgressTokenBackgroundIndexProgress {
			// Clangd background index progress
			bgIdxParams := clangdlsp.BackgroundIndexProgressParams{}
			if err := json.Unmarshal(req.Params(), &bgIdxParams); err != nil {
				return fmt.Errorf("failed to unmarshall background index progress params: %s", err)
			}

			// Send message on channel if the background indexing is complete
			if bgIdxParams.Value.Kind == clangdlsp.BackgroundIndexProgressEnd {
				manager.lspNotifications.backgroundIndexDone <- struct{}{}
			}
		}
	}

	// Reply with null to meat JSON spec
	manager.logger.Debug("received response over connection", zap.Any("req", req))
	return reply(ctx, nil, nil)
}

// NewLSPManager creates a new LSPManager.
// This method spawns a LSP server child process for the LSPManager to use.
func NewLSPManager(ctx context.Context, logger *zap.Logger) (*LSPManager, error) {
	// Start LSP server
	proc, err := cmd.NewCmdCloser(ctx, logger, "clangd", []string{
		//"--log=verbose",
		"--limit-results=0",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to run C LSP: %s", err)
	}

	logger.Info("running lsp")

	stream := jsonrpc2.NewStream(proc)
	conn := jsonrpc2.NewConn(stream)

	server := clangdlsp.NewClangdServer(conn, logger)

	// Create document collection
	docColl := LSPDocumentCollection{
		server:    server.Server,
		documents: []LSPDocument{},
	}

	// Create LSPManager
	manager := LSPManager{
		ctx:              ctx,
		logger:           logger,
		stream:           stream,
		conn:             conn,
		server:           server,
		lspNotifications: NewLSPNotifications(),
		docColl:          docColl,
	}

	// Start goroutine to handle JSON RPC messages
	go func() {
		conn.Go(ctx, manager.HandleLSPMsg)
	}()

	//client := protocol.ClientDispatcher(conn, logger)

	return &manager, nil
}
