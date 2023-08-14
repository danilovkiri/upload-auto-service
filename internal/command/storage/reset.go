// Package storage provides CLI commands definitions and execution logic.

package storage

import (
	"fmt"
	"upload-service-auto/internal/storage/v1/psql"
	"upload-service-auto/internal/syncutils"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// ResetCommand defines a new command struct and sets its attributes.
type ResetCommand struct {
	log       *zerolog.Logger
	storage   *psql.Storage
	syncUtils *syncutils.SyncUtils
}

// NewResetCommand creates a new command instance.
func NewResetCommand(
	logger *zerolog.Logger,
	storage *psql.Storage,
	syncUtils *syncutils.SyncUtils,
) *ResetCommand {
	logger.Debug().Msg("calling initializer of storage:reset command")
	return &ResetCommand{
		log:       logger,
		storage:   storage,
		syncUtils: syncUtils,
	}
}

// Describe handles command description when invoked.
func (t *ResetCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "storage",
		Name:     "storage:reset",
		Usage:    "Reset DB to a virgin state",
		Action:   t.Execute,
	}
}

// Execute runs the command-associated execution logic.
func (t *ResetCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "storage:reset"
		handlerKey = "cli_command"
	)

	defer func() {
		t.syncUtils.SyncCancel()
		t.syncUtils.Wg.Wait()
	}()

	t.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	err := t.storage.DropAll()
	if err != nil {
		t.log.Fatal().Err(err).Msg("could not perform DB drop")
	}

	return nil
}
