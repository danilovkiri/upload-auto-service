// Package messenger provides CLI commands definitions and execution logic.

package messenger

import (
	"encoding/json"
	"fmt"
	busamqp "upload-service-auto/internal/bus/amqp"
	"upload-service-auto/internal/bus/errors"
	"upload-service-auto/internal/bus/modelbus"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/syncutils"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// CreateCommand defines a new command struct and sets its attributes.
type CreateCommand struct {
	log       *zerolog.Logger
	cfg       *config.Config
	amqp      *busamqp.AMQP
	syncUtils *syncutils.SyncUtils
}

// NewCreateCommand creates a new command instance.
func NewCreateCommand(
	logger *zerolog.Logger,
	cfg *config.Config,
	amqp *busamqp.AMQP,
	syncUtils *syncutils.SyncUtils,
) *CreateCommand {
	logger.Debug().Msg("calling initializer of messenger:create command")
	return &CreateCommand{
		log:       logger,
		cfg:       cfg,
		amqp:      amqp,
		syncUtils: syncUtils,
	}
}

// Describe handles command description when invoked.
func (t *CreateCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "messenger",
		Name:     "messenger:create",
		Usage:    "Create an invoice for data validation/processing",
		Action:   t.Execute,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "type",
				Usage:    "Invoice type, either `validate`, `process` or `ready`",
				Aliases:  []string{"t"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "response-type",
				Usage:   "Response type, either `validation`, `processing` for `ready` message.",
				Aliases: []string{"r"},
			},
			&cli.StringFlag{
				Name:     "user-id",
				Usage:    "User identifier (userID)",
				Aliases:  []string{"u"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "barcode",
				Usage:   "Barcode for attachment",
				Aliases: []string{"b"},
			},
			&cli.StringFlag{
				Name:     "file-name",
				Usage:    "Input file name (as stored in S3)",
				Aliases:  []string{"f"},
				Required: true,
			},
		},
	}
}

// Execute runs the command-associated execution logic.
func (t *CreateCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "messenger:create"
		handlerKey = "cli_command"
	)

	var (
		messageType  = ctx.String("type")
		responseType = ctx.String("response-type")
		userID       = ctx.String("user-id")
		fileName     = ctx.String("file-name")
		barcode      = ctx.String("barcode")
	)

	defer func() {
		t.syncUtils.SyncCancel()
		t.syncUtils.Wg.Wait()
	}()

	t.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	switch messageType {
	case "ready":
		if responseType == "" {
			return fmt.Errorf("string flag `--response-type` is required for `%s` message type", messageType)
		}
		msg := modelbus.Rsp{
			UserID:   userID,
			FileName: fileName,
			RspType:  responseType,
			IsReady:  true,
		}
		serialized, err := json.Marshal(msg)
		if err != nil {
			t.log.Error().Err(err).Msg(errors.AMQPMarshallingError)
			return err
		}
		publishing := amqp.Publishing{
			ContentType: "application/json",
			Headers:     amqp.Table{},
			Body:        serialized,
		}
		var exchName string
		if responseType == "validation" {
			exchName = t.cfg.AMQP.ValidationExchangeOutputName
		} else if responseType == "processing" {
			exchName = t.cfg.AMQP.ProcessingExchangeOutputName
		}
		err = t.amqp.PublishToExchange(exchName, publishing)
		if err != nil {
			t.log.Error().Err(err).Msg(errors.AMQPSendingError)
			return err
		}
		t.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("AMQP: message was published to %s", exchName))
	case "validate":
		msg := modelbus.MsgValidate{
			UserID:   userID,
			FileName: fileName,
		}
		serialized, err := json.Marshal(msg)
		if err != nil {
			t.log.Error().Err(err).Msg(errors.AMQPMarshallingError)
			return err
		}
		publishing := amqp.Publishing{
			ContentType: "application/json",
			Headers:     amqp.Table{},
			Body:        serialized,
		}
		err = t.amqp.PublishToExchange(t.cfg.AMQP.ValidationExchangeInputName, publishing)
		if err != nil {
			t.log.Error().Err(err).Msg(errors.AMQPSendingError)
			return err
		}
		t.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("AMQP: message was published to %s", t.cfg.AMQP.ValidationExchangeInputName))
	case "process":
		if barcode == "" {
			return fmt.Errorf("string flag `--barcode` is required for `%s` message type", messageType)
		}
		msg := modelbus.MsgProcess{
			UserID:   userID,
			FileName: fileName,
			Barcode:  barcode,
		}
		serialized, err := json.Marshal(msg)
		if err != nil {
			t.log.Error().Err(err).Msg(errors.AMQPMarshallingError)
			return err
		}
		publishing := amqp.Publishing{
			ContentType: "application/json",
			Headers:     amqp.Table{},
			Body:        serialized,
		}
		err = t.amqp.PublishToExchange(t.cfg.AMQP.ProcessingExchangeInputName, publishing)
		if err != nil {
			t.log.Error().Err(err).Msg(errors.AMQPSendingError)
			return err
		}
		t.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("AMQP: message was published to %s", t.cfg.AMQP.ProcessingExchangeInputName))
	default:
		return fmt.Errorf("invalid invoice type %s", messageType)
	}
	return nil
}
