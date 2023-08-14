// Package dig implements logic for dependency injection using uber-go/dig.

package dig

import (
	"fmt"
	"upload-service-auto/internal/agent/agent"
	"upload-service-auto/internal/api/v1/rest/handlers"
	"upload-service-auto/internal/bus/amqp"
	amqpHandlers "upload-service-auto/internal/bus/handlers"
	cli2 "upload-service-auto/internal/cli"
	"upload-service-auto/internal/command"
	commandFile "upload-service-auto/internal/command/file"
	commandHTTP "upload-service-auto/internal/command/http"
	commandMessenger "upload-service-auto/internal/command/messenger"
	commandStorage "upload-service-auto/internal/command/storage"
	commandUser "upload-service-auto/internal/command/user"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/logger"
	"upload-service-auto/internal/processor/v1/processor"
	"upload-service-auto/internal/productmanager"
	"upload-service-auto/internal/s3/s3"
	"upload-service-auto/internal/storage/v1/psql"
	"upload-service-auto/internal/syncutils"

	"go.uber.org/dig"
)

var definitions = []interface{}{
	handlers.NewEndpointHandlers,
	commandFile.NewProcessCommand,
	commandFile.NewValidateCommand,
	commandHTTP.NewServeCommand,
	commandStorage.NewMigrateCommand,
	commandStorage.NewResetCommand,
	commandUser.NewResetCommand,
	commandUser.NewDeleteCommand,
	commandUser.NewInfoCommand,
	commandUser.NewAllCommand,
	commandMessenger.NewConsumeCommand,
	commandMessenger.NewCreateCommand,
	config.NewConfig,
	logger.NewLog,
	processor.NewProcessor,
	productmanager.NewProductManager,
	s3.NewService,
	psql.NewStorage,
	cli2.NewApp,
	syncutils.NewSyncUtils,
	amqp.NewAMQP,
	amqpHandlers.NewAMQPHandler,
	agent.NewAgent,
}

func buildContainer() (*dig.Container, error) {
	container := dig.New()

	for _, definition := range definitions {
		if err := container.Provide(definition); err != nil {
			return nil, fmt.Errorf("failed to provide service: %w", err)
		}
	}

	if err := commands(container); err != nil {
		return nil, fmt.Errorf("failed to provide commands: %w", err)
	}

	return container, nil
}

func commands(container *dig.Container) error {
	if err := container.Provide(func(
		httpServeCommand *commandHTTP.ServeCommand,
		fileValidateCommand *commandFile.ValidateCommand,
		fileProcessCommand *commandFile.ProcessCommand,
		migrateCommand *commandStorage.MigrateCommand,
		storageResetCommand *commandStorage.ResetCommand,
		userResetCommand *commandUser.ResetCommand,
		userDeleteCommand *commandUser.DeleteCommand,
		userInfoCommand *commandUser.InfoCommand,
		userAllCommand *commandUser.AllCommand,
		consumeCommand *commandMessenger.ConsumeCommand,
		createCommand *commandMessenger.CreateCommand,

	) []command.Command {
		return []command.Command{
			httpServeCommand,
			fileValidateCommand,
			fileProcessCommand,
			migrateCommand,
			storageResetCommand,
			userResetCommand,
			userDeleteCommand,
			userInfoCommand,
			userAllCommand,
			consumeCommand,
			createCommand,
		}
	}); err != nil {
		return fmt.Errorf("failed to define application: %w", err)
	}

	return nil
}
