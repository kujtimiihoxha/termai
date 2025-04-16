package app

import (
	"context"
	"time"

	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/logging"
	"github.com/kujtimiihoxha/termai/internal/lsp"
	"github.com/kujtimiihoxha/termai/internal/lsp/watcher"
)

func (app *App) initLSPClients(ctx context.Context) {
	cfg := config.Get()

	// Initialize LSP clients
	for name, clientConfig := range cfg.LSP {
		app.createAndStartLSPClient(ctx, name, clientConfig.Command, clientConfig.Args...)
	}
}

// createAndStartLSPClient creates a new LSP client, initializes it, and starts its workspace watcher
func (app *App) createAndStartLSPClient(ctx context.Context, name string, command string, args ...string) {
	// Create a specific context for initialization with a timeout

	// Create the LSP client
	lspClient, err := lsp.NewClient(ctx, command, args...)
	if err != nil {
		logging.Error("Failed to create LSP client for", name, err)
		return

	}

	initCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// Initialize with the initialization context
	_, err = lspClient.InitializeLSPClient(initCtx, config.WorkingDirectory())
	if err != nil {
		logging.Error("Initialize failed", "name", name, "error", err)
		// Clean up the client to prevent resource leaks
		lspClient.Close()
		return
	}

	// Create a child context that can be canceled when the app is shutting down
	watchCtx, cancelFunc := context.WithCancel(ctx)
	workspaceWatcher := watcher.NewWorkspaceWatcher(lspClient)

	// Store the cancel function to be called during cleanup
	app.cancelFuncsMutex.Lock()
	app.watcherCancelFuncs = append(app.watcherCancelFuncs, cancelFunc)
	app.cancelFuncsMutex.Unlock()

	// Add the watcher to a WaitGroup to track active goroutines
	app.watcherWG.Add(1)

	// Add to map with mutex protection before starting goroutine
	app.clientsMutex.Lock()
	app.LSPClients[name] = lspClient
	app.clientsMutex.Unlock()

	go app.runWorkspaceWatcher(watchCtx, name, workspaceWatcher)
}

// runWorkspaceWatcher executes the workspace watcher for an LSP client
func (app *App) runWorkspaceWatcher(ctx context.Context, name string, workspaceWatcher *watcher.WorkspaceWatcher) {
	defer app.watcherWG.Done()
	defer logging.RecoverPanic("LSP-"+name, func() {
		// Try to restart the client
		app.restartLSPClient(ctx, name)
	})

	workspaceWatcher.WatchWorkspace(ctx, config.WorkingDirectory())
	logging.Info("Workspace watcher stopped", "client", name)
}

// restartLSPClient attempts to restart a crashed or failed LSP client
func (app *App) restartLSPClient(ctx context.Context, name string) {
	// Get the original configuration
	cfg := config.Get()
	clientConfig, exists := cfg.LSP[name]
	if !exists {
		logging.Error("Cannot restart client, configuration not found", "client", name)
		return
	}

	// Clean up the old client if it exists
	app.clientsMutex.Lock()
	oldClient, exists := app.LSPClients[name]
	if exists {
		delete(app.LSPClients, name) // Remove from map before potentially slow shutdown
	}
	app.clientsMutex.Unlock()

	if exists && oldClient != nil {
		// Try to shut it down gracefully, but don't block on errors
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = oldClient.Shutdown(shutdownCtx)
		cancel()
	}

	// Create a new client using the shared function
	app.createAndStartLSPClient(ctx, name, clientConfig.Command, clientConfig.Args...)
	logging.Info("Successfully restarted LSP client", "client", name)
}
