// Package file provides CLI commands definitions and execution logic.

package file

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"upload-service-auto/internal/agent/agent"
	"upload-service-auto/internal/command/errors"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/syncutils"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// ValidateCommand defines a new command struct and sets its attributes.
type ValidateCommand struct {
	log       *zerolog.Logger
	cfg       *config.Config
	syncUtils *syncutils.SyncUtils
	agent     *agent.Agent
}

// NewValidateCommand creates a new command instance.
func NewValidateCommand(
	logger *zerolog.Logger,
	cfg *config.Config,
	syncUtils *syncutils.SyncUtils,
	agent *agent.Agent,
) *ValidateCommand {
	logger.Debug().Msg("calling initializer of file:validate command")
	return &ValidateCommand{
		log:       logger,
		cfg:       cfg,
		syncUtils: syncUtils,
		agent:     agent,
	}
}

// Describe handles command description when invoked.
func (t *ValidateCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "file",
		Name:     "file:validate",
		Usage:    "Accept user-file pair, run validation and save its results to DB",
		Action:   t.Execute,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user-id",
				Usage:    "User identifier (userID)",
				Aliases:  []string{"u"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "file-path",
				Usage:    "Input file path",
				Aliases:  []string{"f"},
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Disables interaction with DB, only performs validation",
				Value: false,
			},
		},
	}
}

// Execute runs the command-associated execution logic.
func (t *ValidateCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "file:validate"
		handlerKey = "cli_command"
		userIDKey  = "userID"
		fromQueue  = false
	)

	var (
		userID   = ctx.String("user-id")
		filePath = ctx.String("file-path")
		dryRun   = ctx.Bool("dry-run")
	)

	t.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	ctxMain, cancel := context.WithTimeout(t.syncUtils.Ctx, 60*time.Second)
	defer func() {
		cancel()
		t.syncUtils.SyncCancel()
		t.syncUtils.Wg.Wait()
	}()

	filePathSplit := strings.Split(filePath, "/")
	fileName := filePathSplit[len(filePathSplit)-1]

	f, err := os.ReadFile(filePath)
	if err != nil {
		t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.FileReadingError)
		return err
	}

	tempFileRelName := uuid.New().String() + "_" + fileName

	tempFile, err := os.Create(t.cfg.Docker.MountDir + "/source/" + tempFileRelName)
	defer func(tempFile *os.File) {
		err := tempFile.Close()
		if err != nil {
			panic(err)
		}
	}(tempFile)
	if err != nil {
		t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.TempFileOpeningError)
		return err
	}

	_, err = tempFile.Write(f)
	if err != nil {
		t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.TempFileWritingError)
		return err
	}

	validationData, err := t.agent.Validate(ctxMain, userID, tempFileRelName, handler, dryRun, fromQueue)
	if err != nil {
		t.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.ValidationRunError)
		return err
	}

	t.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Dict("validation_data", zerolog.Dict().Str("mode", validationData.Mode).Str("sex", validationData.Sex).Str("error", validationData.Err).Bool("passed", validationData.Passed)).Msg("validation is complete")
	return nil
}
