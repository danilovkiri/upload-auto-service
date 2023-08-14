// Package command provides CLI commands definitions and execution logic.

package command

import "github.com/urfave/cli/v2"

// Command defines an interface for usage in all CLI commands inside the app.
type Command interface {
	Describe() *cli.Command
	Execute(*cli.Context) error
}
