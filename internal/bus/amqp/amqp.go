// Package amqp implements AMQP service.

package amqp

import (
	"context"
	"encoding/json"
	"upload-service-auto/internal/bus/errors"
	"upload-service-auto/internal/bus/modelbus"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/syncutils"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

// AMQP defines queue client object and sets its attributes.
type AMQP struct {
	config          *config.Config
	log             *zerolog.Logger
	channel         *amqp.Channel
	validationQueue *amqp.Queue
	processingQueue *amqp.Queue
	rrsQueue        *amqp.Queue
	syncUtils       *syncutils.SyncUtils
}

// NewAMQP initializes a new AMQP service.
func NewAMQP(config *config.Config, logger *zerolog.Logger, syncUtils *syncutils.SyncUtils) *AMQP {
	logger.Debug().Msg("calling initializer of AMQP service")
	t := &AMQP{
		config:    config,
		log:       logger,
		syncUtils: syncUtils,
	}
	if err := t.init(); err != nil {
		t.log.Fatal().Err(err).Msg(errors.AMQPInitiationError)
	}
	return t
}

// init performs declaration and bindings of queues and exchanges.
func (a *AMQP) init() error {
	a.log.Debug().Msg("calling `init` method")
	conn, err := amqp.Dial(a.config.AMQP.Addr)
	if err != nil {
		a.log.Error().Err(err).Msg(errors.AMQPConnectionError)
		return err
	}

	channel, err := conn.Channel()
	a.channel = channel
	if err != nil {
		a.log.Error().Err(err).Msg(errors.AMQPChannelOpeningError)
		return err
	}

	if err = channel.Qos(1, 0, false); err != nil {
		a.log.Error().Err(err).Msg(errors.AMQPSettingQosError)
		return err
	}

	var (
		validationQueue amqp.Queue
		processingQueue amqp.Queue
		rrsQueue        amqp.Queue
		waitGroup       errgroup.Group
	)

	{ // exchange declaration
		waitGroup.Go(func() error {
			if err = channel.ExchangeDeclare(a.config.AMQP.ValidationExchangeInputName,
				"fanout", true, false, false, false, nil); err != nil {
				return err
			}
			return nil
		})
		waitGroup.Go(func() error {
			if err = channel.ExchangeDeclare(a.config.AMQP.ValidationExchangeOutputName,
				"fanout", true, false, false, false, nil); err != nil {
				return err
			}
			return nil
		})
		waitGroup.Go(func() error {
			if err = channel.ExchangeDeclare(a.config.AMQP.ProcessingExchangeInputName,
				"fanout", true, false, false, false, nil); err != nil {
				return err
			}
			return nil
		})
		waitGroup.Go(func() error {
			if err = channel.ExchangeDeclare(a.config.AMQP.ProcessingExchangeOutputName,
				"fanout", true, false, false, false, nil); err != nil {
				return err
			}
			return nil
		})
		if err := waitGroup.Wait(); err != nil {
			a.log.Error().Err(err).Msg(errors.AMQPExchangeDeclarationError)
			return err
		}
	}

	{ // queue declaration
		waitGroup.Go(func() error {
			if validationQueue, err = channel.QueueDeclare(a.config.AMQP.ValidationQueueName,
				false, false, false, false, amqp.Table{}); err != nil {
				return err
			}
			a.validationQueue = &validationQueue
			return nil
		})
		waitGroup.Go(func() error {
			if processingQueue, err = channel.QueueDeclare(a.config.AMQP.ProcessingQueueName,
				false, false, false, false, amqp.Table{
					"x-consumer-timeout": 21600000,
				}); err != nil {
				return err
			}
			a.processingQueue = &processingQueue
			return nil
		})
		waitGroup.Go(func() error {
			if rrsQueue, err = channel.QueueDeclare(a.config.AMQP.RRSQueueName,
				false, false, false, false, amqp.Table{}); err != nil {
				return err
			}
			a.rrsQueue = &rrsQueue
			return nil
		})
		if err := waitGroup.Wait(); err != nil {
			a.log.Error().Err(err).Msg(errors.AMQPQueueDeclarationError)
			return err
		}
	}

	{ // queue binding
		waitGroup.Go(func() error {
			if err = channel.QueueBind(processingQueue.Name,
				"", a.config.AMQP.ProcessingExchangeInputName, false, nil); err != nil {
				return err
			}
			return nil
		})
		waitGroup.Go(func() error {
			if err = channel.QueueBind(validationQueue.Name,
				"", a.config.AMQP.ValidationExchangeInputName, false, nil); err != nil {
				return err
			}
			return nil
		})
		waitGroup.Go(func() error {
			if err = channel.QueueBind(rrsQueue.Name,
				"", a.config.AMQP.ValidationExchangeOutputName, false, nil); err != nil {
				return err
			}
			return nil
		})
		waitGroup.Go(func() error {
			if err = channel.QueueBind(rrsQueue.Name,
				"", a.config.AMQP.ProcessingExchangeOutputName, false, nil); err != nil {
				return err
			}
			return nil
		})
	}

	a.syncUtils.Wg.Add(1)
	go func() {
		defer a.syncUtils.Wg.Done()
		<-a.syncUtils.Ctx.Done()
		err = conn.Close()
		if err != nil {
			a.log.Fatal().Err(err).Msg("could not close AMQP connection")
		}
		a.log.Debug().Msg("AMQP connection was closed")
	}()
	return nil
}

// PublishToExchange publishes a message to the specified exchange.
func (a *AMQP) PublishToExchange(exchange string, msg amqp.Publishing) error {
	a.log.Debug().Msg("calling `PublishToExchange` method")

	if err := a.channel.PublishWithContext(a.syncUtils.Ctx, exchange, "", false, false, msg); err != nil {
		a.log.Error().Err(err).Msg(errors.AMQPPublishingError)
		return err
	}

	a.log.Info().Msg("message was successfully published to AMQP")

	return nil
}

// AddInterpretationQueueListener is a middleware method for handling different AMQP handlers.
func (a *AMQP) AddInterpretationQueueListener(ctx context.Context, republish bool, queueName, exchangeName, exchangeNameOut, runType string, fn func(ctx context.Context, d *amqp.Delivery) (string, string, bool, error)) error {
	messages, err := a.channel.Consume(queueName,
		"", false, false, false, false, nil)
	if err != nil {
		a.log.Error().Err(err).Msg(errors.AMQPConsumingError)
		return err
	}

	var waitGroup errgroup.Group
	waitGroup.Go(func() error {
		for delivery := range messages {
			a.log.Debug().Str("body", string(delivery.Body)).Msg("AMQP: received message")

			userID, fileName, status, fnErr := fn(ctx, &delivery)
			if fnErr == nil {
				if ackErr := delivery.Ack(false); ackErr != nil {
					a.log.Error().Err(err).Msg(errors.AMQPAckError)
					return err
				}
			} else {
				a.log.Warn().Msg(errors.AMQPMessageProcessingError)
				if ackErr := delivery.Ack(false); ackErr != nil {
					a.log.Error().Err(err).Msg(errors.AMQPAckError)
					return err
				}

				if republish {
					retryMsg := amqp.Publishing{
						ContentType: delivery.ContentType,
						Headers:     delivery.Headers,
						Body:        delivery.Body,
					}

					err := a.PublishToExchange(exchangeName, retryMsg)
					if err != nil {
						a.log.Error().Err(err).Msg(errors.AMQPSendingError)
						return err
					}
				}
			}

			// send status to rrs
			msg := modelbus.Rsp{
				UserID:   userID,
				FileName: fileName,
				RspType:  runType,
				IsReady:  status,
			}

			serialized, err := json.Marshal(msg)
			if err != nil {
				a.log.Error().Err(err).Msg(errors.AMQPMarshallingError)
				return err
			}

			publishing := amqp.Publishing{
				ContentType: "application/json",
				Headers:     amqp.Table{},
				Body:        serialized,
			}
			err = a.PublishToExchange(exchangeNameOut, publishing)
			if err != nil {
				a.log.Error().Err(err).Msg(errors.AMQPSendingError)
				return err
			}

		}
		return nil
	})

	a.log.Info().Msg("AMQP: consumer started")

	if err := waitGroup.Wait(); err != nil {
		a.log.Error().Err(err).Msg(errors.AMQPListeningError)
		return err
	}

	return nil
}
