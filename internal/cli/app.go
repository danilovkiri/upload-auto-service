// Package cli provides a method for new application instantiation.

package cli

import (
	"upload-service-auto/internal/command"

	"github.com/urfave/cli/v2"
)

// NewApp initializes a new cli.App service.
func NewApp(definitions []command.Command) *cli.App {
	commands := make([]*cli.Command, 0, len(definitions))

	for _, definition := range definitions {
		commands = append(commands, definition.Describe())
	}

	return &cli.App{
		Name:     "Upload Service Processor",
		Version:  "0.0.1",
		Commands: commands,
	}
}
