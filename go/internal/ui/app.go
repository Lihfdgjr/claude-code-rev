package ui

import (
	"context"

	"claudecode/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// App is the top-level UI entry point. It owns the Bubble Tea program and
// the model that talks to the chat Driver.
type App struct {
	driver core.Driver
	model  *Model
}

// New constructs an App bound to the given driver.
func New(driver core.Driver) *App {
	return &App{
		driver: driver,
		model:  NewModel(driver),
	}
}

// Run starts the Bubble Tea event loop and blocks until the user quits or
// the supplied context is cancelled. Returns whatever error the program
// produced (nil on a clean quit).
func (a *App) Run(ctx context.Context) error {
	prog := tea.NewProgram(
		a.model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	// Cancel any in-flight turn when the context is torn down so we don't
	// leak a producer goroutine pushing into an event channel nobody reads.
	go func() {
		<-ctx.Done()
		a.driver.Cancel()
	}()

	_, err := prog.Run()
	if err != nil && ctx.Err() != nil {
		// Suppress the program error if the cause was context cancellation;
		// surface the context error instead.
		return ctx.Err()
	}
	return err
}
