// Package dig implements logic for dependency injection using uber-go/dig.

package dig

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

type App struct {
	Kernel *Kernel
}

func (t *App) Boot() error {
	if err := t.Kernel.Build(); err != nil {
		return fmt.Errorf("failed to build kernel: %w", err)
	}

	return nil
}

func (t *App) Run() error {
	if err := t.Boot(); err != nil {
		return err
	}

	if err := t.Kernel.Container.Invoke(func(
		app *cli.App,
	) error {
		if err := app.Run(os.Args); err != nil {
			return fmt.Errorf("failed to run application: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func NewApp(kernel *Kernel) *App {
	return &App{
		Kernel: kernel,
	}
}
