package clangdlsp

import (
	"context"
	"fmt"

	"github.com/segmentio/encoding/json"
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

	InitializationOptions InitializationOptions `json:"initializationOptions"`
}

type InitializationOptions struct {
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
	// Make request
	var ifaceRes interface{}
	res := InitializeResult{}

	server.logger.Debug("call "+protocol.MethodInitialize, zap.Any("params", params))
	err := protocol.Call(ctx, server.conn, protocol.MethodInitialize, params, &ifaceRes)
	if err != nil {
		return nil, err
	}

	// Deserialize into embedded struct
	marshalledRes, err := json.Marshal(ifaceRes)
	if err != nil {
		return nil, fmt.Errorf("failed to re-marshall response: %s", err)
	}
	if err = json.Unmarshal(marshalledRes, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshall response into base struct: %s", err)
	}
	if err = json.Unmarshal(marshalledRes, &(res.InitializeResult)); err != nil {
		return nil, fmt.Errorf("failed to unmarshall response into embedded struct: %s", err)
	}
	return &res, nil
}
