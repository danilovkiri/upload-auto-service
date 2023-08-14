// Package config provides types for handling configuration parameters.

package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// S3Storage defines variables for a subset of configuration parameters.
type S3Storage struct {
	AccessKeyID           string `env:"S3_ACCESS_KEY_ID"`
	SecretAccessKey       string `env:"S3_SECRET_ACCESS_KEY"`
	Endpoint              string `env:"S3_ENDPOINT" env-default:"storage.yandexcloud.net"`
	Region                string `env:"S3_REGION" env-default:"ru-central1"`
	Bucket                string `env:"S3_BUCKET" env-default:"atlas-ru-samples-dev-moscow-array"`
	FolderInternal        string `env:"S3_FOLDER_INTERNAL" env-default:"atlas_raw_data_extended"`
	FolderExternal        string `env:"S3_FOLDER_EXTERNAL" env-default:"external_raw_data"`
	FolderBinary          string `env:"S3_FOLDER_BINARY" env-default:"binary"`
	BucketUpload          string `env:"S3_BUCKET_UPLOAD" env-default:"atlas-ru-samples-dev-moscow-array"`
	FolderUpload          string `env:"S3_FOLDER_UPLOAD" env-default:"upload"`
	AccessKeyIDUpload     string `env:"S3_ACCESS_KEY_ID_UPLOAD"`
	SecretAccessKeyUpload string `env:"S3_SECRET_ACCESS_KEY_UPLOAD"`
}

// Docker defines variables for a subset of configuration parameters.
type Docker struct {
	DockerImageName  string `env:"DOCKER_IMAGE_NAME" env-default:"upload_app:latest"`
	MountDir         string `env:"DOCKER_MOUNT_DIR" env-default:"/mnt"`
	DockerExecutable string `env:"DOCKER_EXEC"`
}

// Server defines variables for a subset of configuration parameters.
type Server struct {
	ServerAddress string        `env:"SERVER_ADDRESS" env-default:":8080"`
	IdleTimeout   time.Duration `env:"IDLE_TIMEOUT" env-default:"120s"`
	ReadTimeout   time.Duration `env:"READ_TIMEOUT" env-default:"120s"`
	WriteTimeout  time.Duration `env:"WRITE_TIMEOUT" env-default:"120s"`
}

// AMQP defines variables for a subset of configuration parameters.
type AMQP struct {
	Addr                         string `env:"AMQP_ADDR"`
	ValidationExchangeInputName  string `env:"AMQP_VALIDATION_EXCHANGE_INPUT_NAME" env-default:"validation_exchange_input"`
	ValidationExchangeOutputName string `env:"AMQP_VALIDATION_EXCHANGE_OUTPUT_NAME" env-default:"validation_exchange_output"`
	ProcessingExchangeInputName  string `env:"AMQP_PROCESSING_EXCHANGE_INPUT_NAME" env-default:"processing_exchange_input"`
	ProcessingExchangeOutputName string `env:"AMQP_PROCESSING_EXCHANGE_OUTPUT_NAME" env-default:"processing_exchange_input"`
	ValidationQueueName          string `env:"AMQP_VALIDATION_QUEUE_NAME" env-default:"validation"`
	ProcessingQueueName          string `env:"AMQP_PROCESSING_QUEUE_NAME" env-default:"processing"`
	RRSQueueName                 string `env:"AMQP_RRS_QUEUE_NAME" env-default:"rrs"`
}

// Config defines configuration parameters for an app.
type Config struct {
	DB        DB
	Logger    Logger
	Docker    Docker
	S3Storage S3Storage
	Server    Server
	AMQP      AMQP
}

// DB defines variables for a subset of configuration parameters.
type DB struct {
	DatabaseDSN string `env:"DATABASE_DSN" env-default:"postgres://upload_client:12345@localhost:5432/upload_db"`
}

// Logger defines variables for a subset of configuration parameters.
type Logger struct {
	Level int `env:"LOG_LEVEL" env-default:"0"`
}

// NewConfig initializes a new Config instance and parses environment variables.
func NewConfig() *Config {
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		panic(err)
	}
	return &cfg
}
