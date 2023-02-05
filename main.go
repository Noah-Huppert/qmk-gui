package main

import (
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"

	"context"
	"fmt"
	"os"
	"path/filepath"

	"embed"

	"github.com/Noah-Huppert/qmk-gui/clangdlsp"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func main() {
	// Setup context and logger
	ctx := context.Background()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to make logger: %s", err))
	}
	defer logger.Sync()

	// Setup GUI
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "qmk-gui",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}

	// Initialize LSP
	logger.Info("initializing C LSP")

	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatal("failed to get working directory", zap.Error(err))
	}

	qmkFirmwareDir := filepath.Join(cwd, "../qmk_firmware")

	initRes, err := server.Initialize(ctx, &clangdlsp.InitializeParams{
		InitializeParams: protocol.InitializeParams{
			ClientInfo: &protocol.ClientInfo{
				Name:    "qmk-gui",
				Version: "pre-alpha",
			},
			Locale: "en-us",
			Capabilities: protocol.ClientCapabilities{
				Workspace: &protocol.WorkspaceClientCapabilities{
					WorkspaceFolders: true,
					SemanticTokens: &protocol.SemanticTokensWorkspaceClientCapabilities{
						RefreshSupport: true,
					},
					Symbol: &protocol.WorkspaceSymbolClientCapabilities{
						DynamicRegistration: true,
						SymbolKind: &protocol.SymbolKindCapabilities{
							ValueSet: []protocol.SymbolKind{
								protocol.SymbolKindFile,
								protocol.SymbolKindModule,
								protocol.SymbolKindNamespace,
								protocol.SymbolKindPackage,
								protocol.SymbolKindClass,
								protocol.SymbolKindMethod,
								protocol.SymbolKindProperty,
								protocol.SymbolKindField,
								protocol.SymbolKindConstructor,
								protocol.SymbolKindEnum,
								protocol.SymbolKindInterface,
								protocol.SymbolKindFunction,
								protocol.SymbolKindVariable,
								protocol.SymbolKindConstant,
								protocol.SymbolKindString,
								protocol.SymbolKindNumber,
								protocol.SymbolKindBoolean,
								protocol.SymbolKindArray,
								protocol.SymbolKindObject,
								protocol.SymbolKindKey,
								protocol.SymbolKindNull,
								protocol.SymbolKindEnumMember,
								protocol.SymbolKindStruct,
								protocol.SymbolKindEvent,
								protocol.SymbolKindOperator,
								protocol.SymbolKindTypeParameter,
							},
						},
					},
				},
				Window: &protocol.WindowClientCapabilities{
					WorkDoneProgress: true,
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
			ProcessID: int32(proc.Pid()),
		},
		InitializationOptions: clangdlsp.InitializationOptions{
			ClangdFileStatus: true,
		},
	})
	if err != nil {
		logger.Fatal("failed to initialize C LSP", zap.Error(err))
	}

	// Check for required LSP capabilities
	if !initRes.ServerCapabilities.ASTProvider {
		logger.Fatal("LSP server does not have AST capability", zap.Any("initRes", initRes))
	} else {
		logger.Debug("LSP server has AST capability")
	}

	if workspaceSymbolProvider, ok := initRes.InitializeResult.Capabilities.WorkspaceSymbolProvider.(bool); ok {
		if !workspaceSymbolProvider {
			logger.Fatal("LSP server does not have workspace symbols capability", zap.Any("initRes.InitializeResult", initRes.InitializeResult))
		} else {
			logger.Debug("LSP server has workspace symbols capability")
		}
	}

	if err = server.Initialized(ctx, nil); err != nil {
		logger.Fatal("failed to send initialized notification", zap.Error(err))
	}

	logger.Info("initialized C LSP")

	// Open file
	keymapCFilePath := filepath.Join(qmkFirmwareDir, "keyboards/moonlander/keymaps/default/keymap.c")
	keymapCURI := uri.File(keymapCFilePath)
	if err = docColl.Open(ctx, keymapCURI); err != nil {
		logger.Fatal("failed to open keymap.c", zap.Error(err))
	}

	/* link, err := server.DocumentLink(ctx, &protocol.DocumentLinkParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: keymapCURI,
		},
	})
	if err != nil {
		logger.Fatal("failed to get document links", zap.Error(err))
	}

	logger.Debug("document links", zap.Any("link", link)) */
	/* bgIdxTok := protocol.NewProgressToken("backgroundIndexProgress")
	err = client.WorkDoneProgressCreate(ctx, &protocol.WorkDoneProgressCreateParams{
		Token: *bgIdxTok,
	})
	if err != nil {
		logger.Fatal("failed to create background index progress token", zap.Error(err))
	} */

	/* client.Progress(ctx, &protocol.ProgressParams{
		Token: *bgIdxTok,
	}) */

	/* symbols, err := client.DocumentSymbol(ctx, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: keymapCURI,
		},
	}) */
	// Search for symbols
	// Doesn't seem like a blank search can be provided
	<-backgroundIndexDone
	symbols, err := server.Symbols(ctx, &protocol.WorkspaceSymbolParams{
		Query: "",
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{
			WorkDoneToken: protocol.NewProgressToken("symbols"),
		},
	})
	if err != nil {
		logger.Fatal("failed to list symbols", zap.Error(err))
	}

	logger.Info("symbols", zap.Any("symbols", symbols))

	// Cleanup server
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
