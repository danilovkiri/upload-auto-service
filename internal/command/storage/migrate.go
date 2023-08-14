// Package storage provides CLI commands definitions and execution logic.

package storage

import (
	"fmt"
	"upload-service-auto/internal/storage/v1/psql"
	"upload-service-auto/internal/syncutils"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// MigrateCommand defines a new command struct and sets its attributes.
type MigrateCommand struct {
	log       *zerolog.Logger
	storage   *psql.Storage
	syncUtils *syncutils.SyncUtils
}

// NewMigrateCommand creates a new command instance.
func NewMigrateCommand(
	logger *zerolog.Logger,
	storage *psql.Storage,
	syncUtils *syncutils.SyncUtils,
) *MigrateCommand {
	logger.Debug().Msg("calling initializer of storage:migrate command")
	return &MigrateCommand{
		log:       logger,
		storage:   storage,
		syncUtils: syncUtils,
	}
}

// Describe handles command description when invoked.
func (t *MigrateCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "storage",
		Name:     "storage:migrate",
		Usage:    "Run DB migration",
		Action:   t.Execute,
	}
}

// Execute runs the command-associated execution logic.
func (t *MigrateCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "storage:migrate"
		handlerKey = "cli_command"
	)

	defer func() {
		t.syncUtils.SyncCancel()
		t.syncUtils.Wg.Wait()
	}()

	t.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	err := t.storage.Migrate()
	if err != nil {
		t.log.Fatal().Err(err).Msg("could not perform migration")
	}

	t.syncUtils.SyncCancel()
	t.syncUtils.Wg.Wait()

	return nil
}
