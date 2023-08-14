// Package http provides CLI commands definitions and execution logic.

package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	_ "upload-service-auto/docs"
	"upload-service-auto/internal/api/v1/rest/handlers"
	"upload-service-auto/internal/api/v1/rest/middleware"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/syncutils"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/urfave/cli/v2"
)

const DefaultHTTPPort = 8080

// ServeCommand defines a new command struct and sets its attributes.
type ServeCommand struct {
	log              *zerolog.Logger
	cfg              *config.Config
	endpointHandlers *handlers.EndpointHandlers
	syncUtils        *syncutils.SyncUtils
}

// NewServeCommand creates a new command instance.
func NewServeCommand(
	logger *zerolog.Logger,
	cfg *config.Config,
	endpointHandlers *handlers.EndpointHandlers,
	syncUtils *syncutils.SyncUtils,
) *ServeCommand {
	logger.Debug().Msg("calling initializer of http:serve command")
	return &ServeCommand{
		log:              logger,
		cfg:              cfg,
		syncUtils:        syncUtils,
		endpointHandlers: endpointHandlers,
	}
}

// Describe handles command description when invoked.
func (t *ServeCommand) Describe() *cli.Command {
	return &cli.Command{
		Category: "http",
		Name:     "http:serve",
		Usage:    "Start HTTP server for handling GET data retrieval requests",
		Action:   t.Execute,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Usage:   "HTTP port",
				Aliases: []string{"p"},
				Value:   DefaultHTTPPort,
			},
		},
	}
}

// Execute runs the command-associated execution logic.
func (t *ServeCommand) Execute(ctx *cli.Context) error {
	const (
		handler    = "http:server"
		handlerKey = "cli_command"
	)
	t.log.Info().Str(handlerKey, handler).Msg(fmt.Sprintf("CLI: %s endpoint hit", handler))

	addr := net.JoinHostPort("", strconv.Itoa(ctx.Int("port")))
	if addr != t.cfg.Server.ServerAddress {
		t.log.Warn().Str("env address", t.cfg.Server.ServerAddress).Str("kwargs address", addr).Msg("server address override")
	}

	r := chi.NewRouter()
	r.Use(middleware.CompressHandle)
	r.Use(middleware.DecompressHandle)
	r.Get("/api/v1/status/{userID}", t.endpointHandlers.GetProcessingStatusHandle)
	r.Get("/api/v1/product/{userID}", t.endpointHandlers.GetProductCodeHandle)
	r.Mount("/api/v1/doc", httpSwagger.WrapHandler)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		IdleTimeout:  t.cfg.Server.IdleTimeout,
		ReadTimeout:  t.cfg.Server.ReadTimeout,
		WriteTimeout: t.cfg.Server.WriteTimeout,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	t.syncUtils.Wg.Add(1)
	go func() {
		<-done
		t.log.Info().Msg("server shutdown attempted")
		ctxTO, cancelTO := context.WithTimeout(t.syncUtils.Ctx, 5*time.Second)
		defer func() {
			t.syncUtils.SyncCancel()
			t.syncUtils.Wg.Done()
			cancelTO()
		}()
		if err := srv.Shutdown(ctxTO); err != nil {
			log.Fatal().Err(err).Msg("server shutdown failed")
		}
	}()

	t.log.Info().Msg("server start attempted")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		t.log.Fatal().Err(err).Msg("server start failed")
	}

	t.syncUtils.Wg.Wait()

	t.log.Info().Msg("server shutdown succeeded")
	return nil
}
