package lsp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

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
