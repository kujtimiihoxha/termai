package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kujtimiihoxha/opencode/internal/app"
	"github.com/kujtimiihoxha/opencode/internal/config"
	"github.com/kujtimiihoxha/opencode/internal/db"
	"github.com/kujtimiihoxha/opencode/internal/llm/agent"
	"github.com/kujtimiihoxha/opencode/internal/logging"
	"github.com/kujtimiihoxha/opencode/internal/pubsub"
	"github.com/kujtimiihoxha/opencode/internal/tui"
	zone "github.com/lrstanley/bubblezone"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "OpenCode",
	Short: "A terminal ai assistant",
	Long:  `A terminal ai assistant`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If the help flag is set, show the help message
		if cmd.Flag("help").Changed {
			cmd.Help()
			return nil
		}

		// Load the config
		debug, _ := cmd.Flags().GetBool("debug")
		cwd, _ := cmd.Flags().GetString("cwd")
		if cwd != "" {
			err := os.Chdir(cwd)
			if err != nil {
				return fmt.Errorf("failed to change directory: %v", err)
			}
		}
		if cwd == "" {
			c, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %v", err)
			}
			cwd = c
		}
		_, err := config.Load(cwd, debug)
		if err != nil {
			return err
		}

		// Connect DB, this will also run migrations
		conn, err := db.Connect()
		if err != nil {
			return err
		}

		// Create main context for the application
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		app, err := app.New(ctx, conn)
		if err != nil {
			logging.Error("Failed to create app: %v", err)
			return err
		}

		// Set up the TUI
		zone.NewGlobal()
		program := tea.NewProgram(
			tui.New(app),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)

		// Initialize MCP tools in the background
		initMCPTools(ctx, app)

		// Setup the subscriptions, this will send services events to the TUI
		ch, cancelSubs := setupSubscriptions(app)

		// Create a context for the TUI message handler
		tuiCtx, tuiCancel := context.WithCancel(ctx)
		var tuiWg sync.WaitGroup
		tuiWg.Add(1)

		// Set up message handling for the TUI
		go func() {
			defer tuiWg.Done()
			defer logging.RecoverPanic("TUI-message-handler", func() {
				attemptTUIRecovery(program)
			})

			for {
				select {
				case <-tuiCtx.Done():
					logging.Info("TUI message handler shutting down")
					return
				case msg, ok := <-ch:
					if !ok {
						logging.Info("TUI message channel closed")
						return
					}
					program.Send(msg)
				}
			}
		}()

		// Cleanup function for when the program exits
		cleanup := func() {
			// Shutdown the app
			app.Shutdown()

			// Cancel subscriptions first
			cancelSubs()

			// Then cancel TUI message handler
			tuiCancel()

			// Wait for TUI message handler to finish
			tuiWg.Wait()

			logging.Info("All goroutines cleaned up")
		}

		// Run the TUI
		result, err := program.Run()
		cleanup()

		if err != nil {
			logging.Error("TUI error: %v", err)
			return fmt.Errorf("TUI error: %v", err)
		}

		logging.Info("TUI exited with result: %v", result)
		return nil
	},
}

// attemptTUIRecovery tries to recover the TUI after a panic
func attemptTUIRecovery(program *tea.Program) {
	logging.Info("Attempting to recover TUI after panic")

	// We could try to restart the TUI or gracefully exit
	// For now, we'll just quit the program to avoid further issues
	program.Quit()
}

func initMCPTools(ctx context.Context, app *app.App) {
	go func() {
		defer logging.RecoverPanic("MCP-goroutine", nil)

		// Create a context with timeout for the initial MCP tools fetch
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Set this up once with proper error handling
		agent.GetMcpTools(ctxWithTimeout, app.Permissions)
		logging.Info("MCP message handling goroutine exiting")
	}()
}

func setupSubscriber[T any](
	ctx context.Context,
	wg *sync.WaitGroup,
	name string,
	subscriber func(context.Context) <-chan pubsub.Event[T],
	outputCh chan<- tea.Msg,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logging.RecoverPanic(fmt.Sprintf("subscription-%s", name), nil)

		for {
			select {
			case event, ok := <-subscriber(ctx):
				if !ok {
					logging.Info("%s subscription channel closed", name)
					return
				}

				// Convert generic event to tea.Msg if needed
				var msg tea.Msg = event

				// Non-blocking send with timeout to prevent deadlocks
				select {
				case outputCh <- msg:
				case <-time.After(500 * time.Millisecond):
					logging.Warn("%s message dropped due to slow consumer", name)
				case <-ctx.Done():
					logging.Info("%s subscription cancelled", name)
					return
				}
			case <-ctx.Done():
				logging.Info("%s subscription cancelled", name)
				return
			}
		}
	}()
}

func setupSubscriptions(app *app.App) (chan tea.Msg, func()) {
	ch := make(chan tea.Msg, 100)
	// Add a buffer to prevent blocking
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	// Setup each subscription using the helper
	setupSubscriber(ctx, &wg, "logging", logging.Subscribe, ch)
	setupSubscriber(ctx, &wg, "sessions", app.Sessions.Subscribe, ch)
	setupSubscriber(ctx, &wg, "messages", app.Messages.Subscribe, ch)
	setupSubscriber(ctx, &wg, "permissions", app.Permissions.Subscribe, ch)

	// Return channel and a cleanup function
	cleanupFunc := func() {
		logging.Info("Cancelling all subscriptions")
		cancel() // Signal all goroutines to stop

		// Wait with a timeout for all goroutines to complete
		waitCh := make(chan struct{})
		go func() {
			defer logging.RecoverPanic("subscription-cleanup", nil)
			wg.Wait()
			close(waitCh)
		}()

		select {
		case <-waitCh:
			logging.Info("All subscription goroutines completed successfully")
		case <-time.After(5 * time.Second):
			logging.Warn("Timed out waiting for some subscription goroutines to complete")
		}

		close(ch) // Safe to close after all writers are done or timed out
	}
	return ch, cleanupFunc
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("help", "h", false, "Help")
	rootCmd.Flags().BoolP("debug", "d", false, "Debug")
	rootCmd.Flags().StringP("cwd", "c", "", "Current working directory")
}
