package lsp

import (
	"context"

	"github.com/Noah-Huppert/qmk-gui/clangdlsp"
	"go.lsp.dev/jsonrpc2"
)

// LSPManager wraps interactions with the language server.
type LSPManager struct {
	ctx context.Context

	stream jsonrpc2.Stream
	conn   jsonrpc2.Conn
	server clangdlsp.ClangdServer
}
