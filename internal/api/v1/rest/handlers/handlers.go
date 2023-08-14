// Package handlers implements handling functions for HTTP endpoints.

// @title Upload Auto Service REST API
// @desc REST API for user-related data retrieval.
//
// @contact.name Kirill Danilov
// @contact.email danilov@atlasbiomed.com
//
// @ver 1.0.0
// @server https://some.domain.dev.com/api/v1 Production API
// @server https://some.domain.prod.com/api/v1 Development API

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"net/http"
	"time"
	"upload-service-auto/internal/agent/agent"
	"upload-service-auto/internal/api/v1/errors"
	"upload-service-auto/internal/api/v1/modeldto"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/storage/v1/psql"

	"github.com/rs/zerolog"
)

const (
	handlerKey = "handler"
	userIDKey  = "userID"
)

// EndpointHandlers defines URLHandler object structure.
type EndpointHandlers struct {
	log     *zerolog.Logger
	cfg     *config.Config
	storage *psql.Storage
	agent   *agent.Agent
}

// NewEndpointHandlers initializes EndpointHandlers object setting its attributes.
func NewEndpointHandlers(
	cfg *config.Config,
	logger *zerolog.Logger,
	storage *psql.Storage,
	agent *agent.Agent,
) *EndpointHandlers {
	logger.Debug().Msg("calling initializer of HTTP handling service")
	return &EndpointHandlers{cfg: cfg, log: logger, storage: storage, agent: agent}
}

// GetProcessingStatusHandle handles requests to get processing status of a user.
// @summary Get processing status request
// @desc Get processing status for a user ID
// @id getProcessingStatus
// @accept x-www-form-urlencoded
// @produce json
// @param userID path string true "User ID to get status for"
// @success 200 {object} modeldto.ResponseProcessingStatus
// @failure 400 {string} Bad request
// @failure 500 {string} Internal Server Error
// @failure 404 {string} Not found
// @failure 417 {string} Expectation failed
// @router /api/v1/status/{userID} [get]
func (h *EndpointHandlers) GetProcessingStatusHandle(w http.ResponseWriter, r *http.Request) {
	const handler = "get-processing-status"

	h.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("HTTP: %s endpoint hit", handler))

	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()

	userID := chi.URLParam(r, "userID")

	status, httpStatus, errorCode := h.agent.GetProcessingStatus(ctx, userID, handler)
	if status == "" {
		http.Error(w, errorCode, httpStatus)
		return
	}

	responseProcessingStatus := modeldto.ResponseProcessingStatus{Status: status}
	resBody, err := json.Marshal(responseProcessingStatus)
	if err != nil {
		h.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.MarshallingError)
		http.Error(w, errors.MarshallingError, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resBody)
	h.log.Info().Str(handlerKey, handler).Str(userIDKey, userID).Msg("response sent")
}

// GetProductCodeHandle handles requests to get product code of a user.
// @summary Get product code request
// @desc Get product code for a user ID
// @id getProductCode
// @accept x-www-form-urlencoded
// @produce json
// @param userID path string true "User ID to get product code for"
// @success 200 {object} modeldto.ResponseProductCode
// @failure 400 {string} Bad request
// @failure 500 {string} Internal Server Error
// @failure 404 {string} Not found
// @failure 417 {string} Expectation failed
// @router /api/v1/product/{userID} [get]
func (h *EndpointHandlers) GetProductCodeHandle(w http.ResponseWriter, r *http.Request) {
	const handler = "get-product-code"

	h.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("HTTP: %s endpoint hit", handler))

	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()

	userID := chi.URLParam(r, "userID")

	productCode, httpStatus, errorCode := h.agent.GetProductCode(ctx, userID, handler)
	if productCode == "" {
		http.Error(w, errorCode, httpStatus)
		return
	}

	responseProductCode := modeldto.ResponseProductCode{ProductCode: productCode}
	resBody, err := json.Marshal(responseProductCode)
	if err != nil {
		h.log.Error().Err(err).Str(handlerKey, handler).Str(userIDKey, userID).Msg(errors.MarshallingError)
		http.Error(w, errors.MarshallingError, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resBody)
	h.log.Info().Str(handlerKey, handler).Msg("response sent")
}
