package lsp

// LSPNotifications is responsible for communicating to the rest of the system that certain LSP notifications have been received.
type LSPNotifications struct {
	// backgroundIndexDone is a channel which has a message sent on it when the LSP server indicates its background index process is complete.
	backgroundIndexDone chan struct{}
}

// NewLSPNotifications creates a new LSPNotifications structure.
func NewLSPNotifications() LSPNotifications {
	return LSPNotifications{
		backgroundIndexDone: make(chan struct{}),
	}
}

// BackgroundIndexDone returns a channel which receives a notification when the LSP server indicates its background index process is complete.
func (notif LSPNotifications) BackgroundIndexDone() <-chan struct{} {
	return notif.backgroundIndexDone
}
