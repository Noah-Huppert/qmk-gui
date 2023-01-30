package clangdlsp

import (
	"context"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// Language server client for the clangd language server. Implements methods to access LSP extensions which the ClangD server adds.
// ClangD extension: https://clangd.llvm.org/extensions
type ClangDServer struct {
	protocol.Server

	// Connection over which JSON RPC takes place
	conn jsonrpc2.Conn

	// Logger used to output information
	logger *zap.Logger
}

func NewClangDServer(conn jsonrpc2.Conn, logger *zap.Logger) ClangDServer {
	return ClangDServer{
		Server: protocol.ServerDispatcher(conn, logger),
		conn:   conn,
		logger: logger,
	}
}

// / Initialize request parameters with ClangD capabilities
type ClangDInitializeParams struct {
	protocol.InitializeParams

	// Enables the ClangD file status extension
	// https://clangd.llvm.org/extensions#file-status
	ClangdFileStatus bool `json:"clangdFileStatus"`
}

func (server ClangDServer) Initialize(ctx context.Context, params *ClangDInitializeParams) (*protocol.InitializeResult, error) {
	res := protocol.InitializeResult{}
	err := protocol.Call(ctx, server.conn, protocol.MethodInitialize, params, &res)

	if err != nil {
		return nil, err
	}
	return &res, nil
}
