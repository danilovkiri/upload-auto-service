// Package file provides CLI commands definitions and execution logic.

package file

import (
	"context"
	"fmt"
	"time"
	"upload-service-auto/internal/agent/agent"
	"upload-service-auto/internal/command/errors"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/processor/v1/processor"
	"upload-service-auto/internal/storage/v1/psql"
	"upload-service-auto/internal/syncutils"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// ProcessCommand defines a new command struct and sets its attributes.
type ProcessCommand struct {
	log       *zerolog.Logger
	cfg       *config.Config
	storage   *psql.Storage
	proc      *processor.Processor
	syncUtils *syncutils.SyncUtils
	agent     *agent.Agent
}

func NewProcessCommand(
	logger *zerolog.Logger,
	cfg *config.Config,
	storage *psql.Storage,
	proc *processor.Processor,
	syncUtils *syncutils.SyncUtils,
	agent *agent.Agent,
) *ProcessCommand {
	logger.Debug().Msg("calling initializer of file:process command")
	return &ProcessCommand{
		log:       logger,
		cfg:       cfg,
		storage:   storage,
		proc:      proc,
		syncUtils: syncUtils,
		agent:     agent,
	}
}

func (t *ProcessCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "file",
		Name:     "file:process",
		Usage:    "Run processing and save its results to DB",
		Action:   t.Execute,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user-id",
				Usage:    "User identifier (userID)",
				Aliases:  []string{"u"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "barcode",
				Usage:    "Barcode for attachment",
				Aliases:  []string{"b"},
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Disables S3 upload, though requires prior wet-run validation",
				Value: false,
			},
		},
	}
}

func (t *ProcessCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "file:process"
		handlerKey = "cli_command"
		userIDKey  = "userID"
		fromQueue  = false
	)

	var (
		userID  = ctx.String("user-id")
		barcode = ctx.String("barcode")
		dryRun  = ctx.Bool("dry-run")
	)

	t.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	ctxMain, cancel := context.WithTimeout(t.syncUtils.Ctx, 6*time.Hour)
	defer func() {
		cancel()
		t.syncUtils.SyncCancel()
		t.syncUtils.Wg.Wait()
	}()

	err := t.agent.Process(ctxMain, userID, barcode, handler, dryRun, fromQueue)
	if err != nil {
		t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.ProcessingRunError)
		return err
	}

	t.syncUtils.SyncCancel()
	t.syncUtils.Wg.Wait()

	t.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Msg("processing is complete")
	return nil
}
