package app

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/kujtimiihoxha/termai/internal/config"
	"github.com/kujtimiihoxha/termai/internal/db"
	"github.com/kujtimiihoxha/termai/internal/history"
	"github.com/kujtimiihoxha/termai/internal/logging"
	"github.com/kujtimiihoxha/termai/internal/lsp"
	"github.com/kujtimiihoxha/termai/internal/lsp/watcher"
	"github.com/kujtimiihoxha/termai/internal/message"
	"github.com/kujtimiihoxha/termai/internal/permission"
	"github.com/kujtimiihoxha/termai/internal/session"
)

type App struct {
	Context context.Context

	Sessions    session.Service
	Messages    message.Service
	Files       history.Service
	Permissions permission.Service

	LSPClients map[string]*lsp.Client

	ceanups []func()
}

func New(ctx context.Context, conn *sql.DB) *App {
	cfg := config.Get()
	logging.Info("Debug mode enabled")

	q := db.New(conn)
	sessions := session.NewService(ctx, q)
	messages := message.NewService(ctx, q)
	files := history.NewService(ctx, q)

	app := &App{
		Context:     ctx,
		Sessions:    sessions,
		Messages:    messages,
		Files:       files,
		Permissions: permission.NewPermissionService(),
		LSPClients:  make(map[string]*lsp.Client),
	}

	for name, client := range cfg.LSP {
		lspClient, err := lsp.NewClient(client.Command, client.Args...)
		app.ceanups = append(app.ceanups, func() {
			lspClient.Close()
		})
		workspaceWatcher := watcher.NewWorkspaceWatcher(lspClient)
		if err != nil {
			logging.Error("Failed to create LSP client for", name, err)
			continue
		}

		_, err = lspClient.InitializeLSPClient(ctx, config.WorkingDirectory())
		if err != nil {
			logging.Error("Initialize failed", "error", err)
			continue
		}
		go workspaceWatcher.WatchWorkspace(ctx, config.WorkingDirectory())
		app.LSPClients[name] = lspClient
	}
	return app
}

func (a *App) Close() {
	for _, cleanup := range a.ceanups {
		cleanup()
	}
	for _, client := range a.LSPClients {
		client.Close()
	}
	slog.Info("App closed")
}
