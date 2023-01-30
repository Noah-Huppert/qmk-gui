package clangdlsp

import (
	"context"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// Language server client for the clangd language server. Implements methods to access LSP extensions which the ClangD server adds.
// ClangD extension: https://clangd.llvm.org/extensions
type ClangdServer struct {
	protocol.Server

	// Connection over which JSON RPC takes place
	conn jsonrpc2.Conn

	// Logger used to output information
	logger *zap.Logger
}

func NewClangdServer(conn jsonrpc2.Conn, logger *zap.Logger) ClangdServer {
	return ClangdServer{
		Server: protocol.ServerDispatcher(conn, logger),
		conn:   conn,
		logger: logger,
	}
}

// Initialize request parameters with Clangd capabilities.
type InitializeParams struct {
	protocol.InitializeParams

	Capabilities ClientCapabilities `json:"capabilities"`
}

type ClientCapabilities struct {
	// Enables the ClangD file status extension
	// https://clangd.llvm.org/extensions#file-status
	ClangdFileStatus bool `json:"clangdFileStatus"`
}

// Initialize response for Clangd.
type InitializeResult struct {
	protocol.InitializeResult

	ServerCapabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	// Indicates the server has AST support.
	ASTProvider bool `json:"astProvider"`
}

func (server ClangdServer) Initialize(ctx context.Context, params *InitializeParams) (*InitializeResult, error) {
	res := InitializeResult{}
	server.logger.Debug("call "+protocol.MethodInitialize, zap.Any("params", params))
	err := protocol.Call(ctx, server.conn, protocol.MethodInitialize, params, &res)

	if err != nil {
		return nil, err
	}
	return &res, nil
}
