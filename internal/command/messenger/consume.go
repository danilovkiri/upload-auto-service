// Package messenger provides CLI commands definitions and execution logic.

package messenger

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"upload-service-auto/internal/bus/handlers"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/processor/v1/processor"
	"upload-service-auto/internal/storage/v1/psql"
	"upload-service-auto/internal/syncutils"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// ConsumeCommand defines a new command struct and sets its attributes.
type ConsumeCommand struct {
	log       *zerolog.Logger
	cfg       *config.Config
	storage   *psql.Storage
	proc      *processor.Processor
	syncUtils *syncutils.SyncUtils
	handler   *handlers.AMQPHandler
}

// NewConsumeCommand creates a new command instance.
func NewConsumeCommand(
	logger *zerolog.Logger,
	cfg *config.Config,
	storage *psql.Storage,
	proc *processor.Processor,
	syncUtils *syncutils.SyncUtils,
	handler *handlers.AMQPHandler,
) *ConsumeCommand {
	logger.Debug().Msg("calling initializer of messenger:consume command")
	return &ConsumeCommand{
		log:       logger,
		cfg:       cfg,
		storage:   storage,
		proc:      proc,
		syncUtils: syncUtils,
		handler:   handler,
	}
}

// Describe handles command description when invoked.
func (t *ConsumeCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "messenger",
		Name:     "messenger:consume",
		Usage:    "Start AMQP messenger consumer",
		Action:   t.Execute,
	}
}

// Execute runs the command-associated execution logic.
func (t *ConsumeCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "messenger:consume"
		handlerKey = "cli_command"
	)
	t.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	t.syncUtils.Wg.Add(1)
	go func() {
		<-done
		t.log.Info().Msg("AMQP client shutdown attempted")
		t.syncUtils.SyncCancel()
		t.syncUtils.Wg.Done()
	}()

	return t.handler.Handle(t.syncUtils.Ctx)
}
