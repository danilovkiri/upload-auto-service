// Package handlers implements AMQP handling functions.

package handlers

import (
	"context"
	"encoding/json"
	"time"
	"upload-service-auto/internal/agent/agent"
	busamqp "upload-service-auto/internal/bus/amqp"
	"upload-service-auto/internal/bus/errors"
	"upload-service-auto/internal/bus/modelbus"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/syncutils"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

const (
	dryRun              = false
	fromQueue           = true
	republishValidation = false
	runTypeValidation   = "validation"
	republishProcessing = false
	runTypeProcessing   = "processing"
	handlerKey          = "amqp"
	userIDKey           = "userID"
)

// AMQPHandler defines an AMQP handler object and sets its attributes.
type AMQPHandler struct {
	log       *zerolog.Logger
	amqp      *busamqp.AMQP
	cfg       *config.Config
	agent     *agent.Agent
	syncUtils *syncutils.SyncUtils
}

// NewAMQPHandler initializes a new AMQP handling service.
func NewAMQPHandler(logger *zerolog.Logger, agent *agent.Agent, amqp *busamqp.AMQP, cfg *config.Config, syncUtils *syncutils.SyncUtils) *AMQPHandler {
	logger.Debug().Msg("calling initializer of AMQP handling service")
	return &AMQPHandler{
		log:       logger,
		agent:     agent,
		amqp:      amqp,
		cfg:       cfg,
		syncUtils: syncUtils,
	}
}

// handleProcessingQueue handles queue message management for processing tasks.
func (h *AMQPHandler) handleProcessingQueue(ctx context.Context, d *amqp.Delivery) (string, string, bool, error) {
	h.log.Debug().Msg("calling `handleProcessingQueue` method")
	const handler = "process"
	ctxMain, cancel := context.WithTimeout(ctx, 6*time.Hour)
	defer cancel()

	msg := modelbus.MsgProcess{}
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		h.log.Error().Err(err).Msg(errors.AMQPUnmarshallingError)
		return "", "", false, err
	}

	userID := msg.UserID
	fileName := msg.FileName
	barcode := msg.Barcode

	err = h.agent.Process(ctxMain, userID, barcode, handler, dryRun, fromQueue)
	if err != nil {
		h.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.AMQPHandlerProcessingError)
		return userID, fileName, false, err
	}

	h.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Msg("processing is complete")
	return userID, fileName, true, nil

}

// handleValidationQueue handles queue message management for validation tasks.
func (h *AMQPHandler) handleValidationQueue(ctx context.Context, d *amqp.Delivery) (string, string, bool, error) {
	h.log.Debug().Msg("calling `handleValidationQueue` method")
	const handler = "validate"

	ctxMain, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	msg := modelbus.MsgValidate{}
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		h.log.Error().Err(err).Msg(errors.AMQPUnmarshallingError)
		return "", "", false, err
	}

	userID := msg.UserID
	fileName := msg.FileName

	validationData, err := h.agent.Validate(ctxMain, userID, fileName, handler, dryRun, fromQueue)
	if err != nil {
		h.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.AMQPHandlerValidationError)
		return userID, fileName, false, err
	}

	h.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Dict("validation_data", zerolog.Dict().Str("mode", validationData.Mode).Str("sex", validationData.Sex).Str("error", validationData.Err).Bool("passed", validationData.Passed)).Msg("validation is complete")
	return userID, fileName, validationData.Passed, nil
}

// Handle is a master handler starting the sub-handlers.
func (h *AMQPHandler) Handle(ctx context.Context) error {
	h.log.Debug().Msg("calling `Handle` method")
	g := &errgroup.Group{}

	// handling validation queue
	h.syncUtils.Wg.Add(1)
	g.Go(func() error {
		defer h.syncUtils.Wg.Done()
		return h.amqp.AddInterpretationQueueListener(
			ctx,
			republishValidation,
			h.cfg.AMQP.ValidationQueueName,
			h.cfg.AMQP.ValidationExchangeInputName,
			h.cfg.AMQP.ValidationExchangeOutputName,
			runTypeValidation,
			h.handleValidationQueue,
		)
	})

	// handling processing queue
	h.syncUtils.Wg.Add(1)
	g.Go(func() error {
		defer h.syncUtils.Wg.Done()
		return h.amqp.AddInterpretationQueueListener(
			ctx,
			republishProcessing,
			h.cfg.AMQP.ProcessingQueueName,
			h.cfg.AMQP.ProcessingExchangeInputName,
			h.cfg.AMQP.ProcessingExchangeOutputName,
			runTypeProcessing,
			h.handleProcessingQueue,
		)
	})
	if err := g.Wait(); err != nil {
		return err
	}

	h.syncUtils.SyncCancel()
	h.syncUtils.Wg.Wait()

	return nil
}
