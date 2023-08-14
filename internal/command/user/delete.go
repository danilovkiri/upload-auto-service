// Package user provides CLI commands definitions and execution logic.

package user

import (
	"context"
	"fmt"
	"time"
	"upload-service-auto/internal/command/errors"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/storage/v1/psql"
	"upload-service-auto/internal/syncutils"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// DeleteCommand defines a new command struct and sets its attributes.
type DeleteCommand struct {
	log       *zerolog.Logger
	cfg       *config.Config
	storage   *psql.Storage
	syncUtils *syncutils.SyncUtils
}

// NewDeleteCommand creates a new command instance.
func NewDeleteCommand(
	logger *zerolog.Logger,
	cfg *config.Config,
	storage *psql.Storage,
	syncUtils *syncutils.SyncUtils,
) *DeleteCommand {
	logger.Debug().Msg("calling initializer of user:delete command")
	return &DeleteCommand{
		log:       logger,
		cfg:       cfg,
		storage:   storage,
		syncUtils: syncUtils,
	}
}

// Describe handles command description when invoked.
func (t *DeleteCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "user",
		Name:     "user:delete",
		Usage:    "Delete all user-related data",
		Action:   t.Execute,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user-id",
				Usage:    "User identifier (userID)",
				Aliases:  []string{"u"},
				Required: true,
			},
		},
	}
}

// Execute runs the command-associated execution logic.
func (t *DeleteCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "user:delete"
		handlerKey = "cli_command"
		userIDKey  = "userID"
	)

	var (
		userID = ctx.String("user-id")
	)

	t.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	ctxMain, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer func() {
		cancel()
		t.syncUtils.SyncCancel()
		t.syncUtils.Wg.Wait()
	}()

	err := t.storage.CheckUserID(ctxMain, userID)
	if err != nil {
		t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.UserNotFoundError)
		return err
	}

	fileName, err := t.storage.GetFileNameForUser(ctxMain, userID)
	if err != nil {
		t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.FileNotFoundError)
		return err
	}

	err = t.storage.RemoveUserData(ctxMain, userID, fileName)
	if err != nil {
		t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg("data deletion failed")
		return err
	}

	return nil
}
