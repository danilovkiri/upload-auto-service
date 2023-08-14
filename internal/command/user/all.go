// Package user provides CLI commands definitions and execution logic.

package user

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"
	"upload-service-auto/internal/command/errors"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/storage/v1/psql"
	"upload-service-auto/internal/syncutils"

	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// AllCommand defines a new command struct and sets its attributes.
type AllCommand struct {
	log       *zerolog.Logger
	cfg       *config.Config
	storage   *psql.Storage
	syncUtils *syncutils.SyncUtils
}

// NewAllCommand creates a new command instance.
func NewAllCommand(
	logger *zerolog.Logger,
	cfg *config.Config,
	storage *psql.Storage,
	syncUtils *syncutils.SyncUtils,
) *AllCommand {
	logger.Debug().Msg("calling initializer of user:all command")
	return &AllCommand{
		log:       logger,
		cfg:       cfg,
		storage:   storage,
		syncUtils: syncUtils,
	}
}

// Describe handles command description when invoked.
func (t *AllCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "user",
		Name:     "user:all",
		Usage:    "Get current stats for all users",
		Action:   t.Execute,
	}
}

// Execute runs the command-associated execution logic.
func (t *AllCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "user:all"
		handlerKey = "cli_command"
		userIDKey  = "userID"
	)

	t.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	ctxMain, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer func() {
		cancel()
		t.syncUtils.SyncCancel()
		t.syncUtils.Wg.Wait()
	}()

	userIDs, err := t.storage.GetAllUserIDs(ctxMain)
	if err != nil {
		t.log.Error().Err(err).Str(handlerKey, handler).Msg("getting all users failed")
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	for _, userID := range userIDs {
		fileName, err := t.storage.GetFileNameForUser(ctxMain, userID)
		if err != nil {
			t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.FileNotFoundError)
		}

		var valid bool
		err = t.storage.CheckIsValid(ctxMain, fileName)
		if err != nil {
			valid = false
		} else {
			valid = true
		}

		productCode, err := t.storage.GetProductCode(ctxMain, userID)
		if err != nil {
			t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.GettingProductCodeError)
		}

		status, err := t.storage.GetProcessingStatus(ctxMain, fileName)
		if err != nil {
			t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.GettingProcessingStatusError)
		}

		table.SetHeader([]string{
			"User ID",
			"File Name",
			"Valid",
			"Product Code",
			"Processing Status",
		})
		table.Append([]string{
			userID,
			fileName,
			strconv.FormatBool(valid),
			productCode,
			status,
		})
	}

	table.Render()

	return nil
}
