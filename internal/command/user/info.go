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

// InfoCommand defines a new command struct and sets its attributes.
type InfoCommand struct {
	log       *zerolog.Logger
	cfg       *config.Config
	storage   *psql.Storage
	syncUtils *syncutils.SyncUtils
}

// NewInfoCommand creates a new command instance.
func NewInfoCommand(
	logger *zerolog.Logger,
	cfg *config.Config,
	storage *psql.Storage,
	syncUtils *syncutils.SyncUtils,
) *InfoCommand {
	logger.Debug().Msg("calling initializer of user:info command")
	return &InfoCommand{
		log:       logger,
		cfg:       cfg,
		storage:   storage,
		syncUtils: syncUtils,
	}
}

// Describe handles command description when invoked.
func (t *InfoCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "user",
		Name:     "user:info",
		Usage:    "Get current stats for one user",
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
func (t *InfoCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "user:info"
		handlerKey = "cli_command"
		userIDKey  = "userID"
	)

	var (
		userID = ctx.String("user-id")
	)

	t.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	ctxMain, cancel := context.WithTimeout(t.syncUtils.Ctx, 500*time.Millisecond)
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

	table := tablewriter.NewWriter(os.Stdout)
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
	table.Render()

	return nil
}
