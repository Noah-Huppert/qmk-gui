package main

import (
	"errors"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"

	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Noah-Huppert/qmk-gui/cmd"
)

// JSON RPC connection handler.
type ClientHandler struct {
	logger *zap.Logger
}

// A handler for JSON RPC responses which just logs the request.
func (h ClientHandler) Handle(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	h.logger.Debug("received response over connection", zap.Any("req", req))
	return nil
}

// Wraps the LSP did open and did close flow.
type LSPDocument struct {
	server protocol.Server
	uri    uri.URI
}

// Opens a file.
func (doc LSPDocument) Open(ctx context.Context) error {
	// Read file
	fileBytes, err := os.ReadFile(doc.uri.Filename())
	if err != nil {
		return fmt.Errorf("failed to read file contents: %s", err)
	}
	fileContents := bytes.NewBuffer(fileBytes).String()

	// Call LSP open
	err = doc.server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        doc.uri,
			LanguageID: "c",
			Version:    0,
			Text:       fileContents,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to call LSP open: %s", err)
	}

	return nil
}

// Closes a document.
func (doc LSPDocument) Close(ctx context.Context) error {
	err := doc.server.DidClose(ctx, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: doc.uri,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to call LSP close: %s", err)
	}

	return nil
}

// Collection of documents.
type LSPDocumentCollection struct {
	server    protocol.Server
	documents []LSPDocument
}

// Open a file
func (coll LSPDocumentCollection) Open(ctx context.Context, uri uri.URI) error {
	doc := LSPDocument{
		server: coll.server,
		uri:    uri,
	}
	if err := doc.Open(ctx); err != nil {
		return fmt.Errorf("failed to open document: %s", err)
	}

	coll.documents = append(coll.documents, doc)

	return nil
}

func (coll LSPDocumentCollection) CloseAll(ctx context.Context) error {
	errs := []string{}

	for _, doc := range coll.documents {
		if err := doc.Close(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("failed to close %s: %s", doc.uri, err))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ", "))
	}

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
	proc, err := cmd.NewCmdCloser(ctx, logger, "clangd")
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

	//client := protocol.ClientDispatcher(conn, logger)
	server := protocol.ServerDispatcher(conn, logger)
	docColl := LSPDocumentCollection{
		server:    server,
		documents: []LSPDocument{},
	}

	// Initialize LSP
	logger.Info("initializing C LSP")

	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatal("failed to get working directory", zap.Error(err))
	}

	qmkFirmwareDir := filepath.Join(cwd, "../qmk_firmware")

	_, err = server.Initialize(ctx, &protocol.InitializeParams{
		ClientInfo: &protocol.ClientInfo{
			Name:    "qmk-gui",
			Version: "pre-alpha",
		},
		Locale: "en",
		Capabilities: protocol.ClientCapabilities{
			Workspace: &protocol.WorkspaceClientCapabilities{
				WorkspaceFolders: true,
				Symbol: &protocol.WorkspaceSymbolClientCapabilities{
					DynamicRegistration: true,
					SymbolKind: &protocol.SymbolKindCapabilities{
						ValueSet: []protocol.SymbolKind{
							protocol.SymbolKindVariable,
							protocol.SymbolKindEnum,
						},
					},
				},
			},
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Synchronization: &protocol.TextDocumentSyncClientCapabilities{
					DynamicRegistration: true,
				},
				PublishDiagnostics: &protocol.PublishDiagnosticsClientCapabilities{
					RelatedInformation:     true,
					VersionSupport:         true,
					CodeDescriptionSupport: true,
					DataSupport:            true,
				},
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
	if err = server.Initialized(ctx, nil); err != nil {
		logger.Fatal("failed to send initialized notification", zap.Error(err))
	}

	logger.Info("initialized C LSP")

	// Open file
	keymapCFilePath := filepath.Join(qmkFirmwareDir, "keyboards/moonlander/keymaps/default/keymap.c")
	keymapCURI := uri.File(keymapCFilePath)
	docColl.Open(ctx, keymapCURI)

	/* symbols, err := client.DocumentSymbol(ctx, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: keymapCURI,
		},
	}) */
	symbols, err := server.Symbols(ctx, &protocol.WorkspaceSymbolParams{
		Query: "",
	})
	if err != nil {
		logger.Fatal("failed to list symbols", zap.Error(err))
	}

	logger.Info("symbols", zap.Any("symbols", symbols))

	// Cleanup server
	time.Sleep(time.Second * 10)

	if err := docColl.CloseAll(ctx); err != nil {
		logger.Fatal("failed to send close events for documents: %s", zap.Error(err))
	}

	if err = server.Shutdown(ctx); err != nil {
		logger.Fatal("failed to shutdown C LSP", zap.Error(err))
	}

	if err = server.Exit(ctx); err != nil {
		logger.Fatal("failed to exit C LSP", zap.Error(err))
	}
}
