// Package agent provides intermediary functionality for HTTP and AMQP handlers.

package agent

import (
	"context"
	"fmt"
	"net/http"
	"upload-service-auto/internal/agent/errors"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/constants"
	"upload-service-auto/internal/processor/v1/models"
	"upload-service-auto/internal/processor/v1/processor"
	"upload-service-auto/internal/productmanager"
	"upload-service-auto/internal/storage/v1/psql"

	"github.com/rs/zerolog"
)

const (
	handlerKey = "handler"
	userIDKey  = "userID"
)

// Agent defines and Agent object ans sets its attributes.
type Agent struct {
	log     *zerolog.Logger
	cfg     *config.Config
	storage *psql.Storage
	proc    *processor.Processor
	manager *productmanager.ProductManager
}

// NewAgent initializes an Agent object.
func NewAgent(
	logger *zerolog.Logger,
	cfg *config.Config,
	storage *psql.Storage,
	proc *processor.Processor,
	manager *productmanager.ProductManager) *Agent {
	logger.Debug().Msg("calling initializer of agent service")
	return &Agent{
		log:     logger,
		cfg:     cfg,
		storage: storage,
		proc:    proc,
		manager: manager,
	}
}

// GetProcessingStatus queries processing status of a user.
func (a *Agent) GetProcessingStatus(ctx context.Context, userID, handler string) (string, int, string) {
	a.log.Debug().Msg("calling `GetProcessingStatus` method")
	err := a.storage.CheckUserID(ctx, userID)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.UserNotFoundError)
		return "", http.StatusNotFound, errors.UserNotFoundError
	}

	fileName, err := a.storage.GetFileNameForUser(ctx, userID)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.FileNotFoundError)
		return "", http.StatusNotFound, errors.FileNotFoundError
	}

	err = a.storage.CheckIsValid(ctx, fileName)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.InvalidFileError)
		return "", http.StatusExpectationFailed, errors.InvalidFileError
	}

	status, err := a.storage.GetProcessingStatus(ctx, fileName)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.GettingProcessingStatusError)
		return "", http.StatusExpectationFailed, errors.GettingProcessingStatusError
	}

	return status, http.StatusOK, ""
}

// GetProductCode queries a product code of a user.
func (a *Agent) GetProductCode(ctx context.Context, userID, handler string) (string, int, string) {
	a.log.Debug().Msg("calling `GetProductCode` method")
	err := a.storage.CheckUserID(ctx, userID)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.UserNotFoundError)
		return "", http.StatusNotFound, errors.UserNotFoundError
	}

	fileName, err := a.storage.GetFileNameForUser(ctx, userID)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.FileNotFoundError)
		return "", http.StatusNotFound, errors.FileNotFoundError
	}

	err = a.storage.CheckIsValid(ctx, fileName)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.InvalidFileError)
		return "", http.StatusExpectationFailed, errors.InvalidFileError
	}

	productCode, err := a.storage.GetProductCode(ctx, userID)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.GettingProductCodeError)
		return "", http.StatusExpectationFailed, errors.GettingProductCodeError
	}

	return productCode, http.StatusOK, ""
}

// Validate runs data validation.
func (a *Agent) Validate(ctx context.Context, userID, fileName, handler string, dryRun, fromQueue bool) (*models.ValidationData, error) {
	a.log.Debug().Msg("calling `Validate` method")
	var userIsNew bool
	err := a.storage.CheckUserID(ctx, userID)
	if err != nil {
		userIsNew = true
	} else {
		userIsNew = false
	}

	if !dryRun {
		if userIsNew {
			err = a.storage.AddNewUserID(ctx, userID)
			if err != nil {
				a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.AddingUserError)
				return nil, err
			}

			err = a.storage.AddNewUserFilePair(ctx, userID, fileName)
			if err != nil {
				a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.AddingFileError)
				return nil, err
			}

			err = a.storage.AddNewValidationEntry(ctx, fileName)
			if err != nil {
				a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.AddingValidationEntryError)
				return nil, err
			}
		} else {
			err := a.storage.UpdateUserFilePair(ctx, userID, fileName)
			if err != nil {
				a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.AddingFileError)
				return nil, err
			}

			// no need to do anything in the validation DB table
		}

	}

	validationData, err := a.proc.RunValidation(ctx, fileName, dryRun, fromQueue)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.ValidationRunError)
		return nil, err
	}

	if !dryRun {
		productCode := a.manager.GetProductCode(validationData.Mode)
		a.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Msg(fmt.Sprintf("derived product code: %s", productCode))

		if validationData.Passed {
			if userIsNew {
				err = a.storage.AddNewProductCode(ctx, userID, productCode)
				if err != nil {
					a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.AddingProductCodeError)
					return nil, err
				}
			} else {
				err = a.storage.UpdateProductCode(ctx, userID, productCode)
				if err != nil {
					a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.AddingProductCodeError)
					return nil, err
				}
			}
		}
	}
	return validationData, nil
}

// Process runs data processing.
func (a *Agent) Process(ctx context.Context, userID, barcode, handler string, dryRun, fromQueue bool) error {
	a.log.Debug().Msg("calling `Process` method")
	err := a.storage.CheckUserID(ctx, userID)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.UserNotFoundError)
		return err
	}

	fileName, err := a.storage.GetFileNameForUser(ctx, userID)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.FileNotFoundError)
		return err
	}

	err = a.storage.CheckIsValid(ctx, fileName)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.InvalidFileError)
		return err
	}

	status, err := a.storage.GetProcessingStatus(ctx, fileName)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.GettingProcessingStatusError)
		return err
	}

	if status == constants.ProcessingStatusRunning {
		a.log.Warn().Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.ProcessingInProgressError)
		return fmt.Errorf("processing blocked: %s", errors.ProcessingInProgressError)
	} else if status == constants.NA {
		err := a.storage.AddNewProcessingEntry(ctx, fileName, barcode)
		if err != nil {
			a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.AddingProcessingEntryError)
			return err
		}
	}

	err = a.proc.RunProcessing(ctx, fileName, barcode, dryRun, fromQueue)
	if err != nil {
		a.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.ProcessingRunError)
		return err
	}

	return nil
}
