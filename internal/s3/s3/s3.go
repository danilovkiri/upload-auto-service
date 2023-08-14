// Package s3 provides data operation service for S3 storage.

package s3

import (
	"fmt"
	"io"
	"os"
	"path"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/s3/errors"
	"upload-service-auto/internal/syncutils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rs/zerolog"
)

// Service defines a new S3 service and sets its attributes.
type Service struct {
	s3up      *s3manager.Uploader
	s3down    *s3.S3
	cfg       *config.Config
	log       *zerolog.Logger
	syncUtils *syncutils.SyncUtils
}

// NewService initializes a new S3 service.
func NewService(config *config.Config, logger *zerolog.Logger, syncUtils *syncutils.SyncUtils) (*Service, error) {
	logger.Debug().Msg("calling initializer of S3 service")
	sessUp := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			config.S3Storage.AccessKeyID,
			config.S3Storage.SecretAccessKey,
			"",
		),
		Region:   &config.S3Storage.Region,
		Endpoint: &config.S3Storage.Endpoint,
	}))
	uploader := s3manager.NewUploader(sessUp)

	sessDown := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			config.S3Storage.AccessKeyIDUpload,
			config.S3Storage.SecretAccessKeyUpload,
			"",
		),
		Region:   &config.S3Storage.Region,
		Endpoint: &config.S3Storage.Endpoint,
	}))
	downloader := s3.New(sessDown)

	return &Service{
		s3down:    downloader,
		s3up:      uploader,
		cfg:       config,
		log:       logger,
		syncUtils: syncUtils,
	}, nil
}

// UploadFile performs data upload to S3.
func (s *Service) UploadFile(filePath, fileType, fileEndName string) error {
	s.log.Debug().Msg("calling `UploadFile` method")
	s.log.Info().Msg(fmt.Sprintf("uploading file %s of type %s to %s", filePath, fileType, fileEndName))
	f, err := os.Open(filePath)
	if err != nil {
		s.log.Error().Err(err).Msg(errors.FileOpeningError)
		return err
	}
	defer f.Close()

	result, err := s.s3up.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.cfg.S3Storage.Bucket),
		Key:    aws.String(s.path(fileType, fileEndName)),
		Body:   f,
	})
	if err != nil {
		s.log.Error().Err(err).Msg(errors.FileUploadError)
		return err
	}
	s.log.Info().Msg(fmt.Sprintf("file uploaded to, %s\\n", result.Location))
	return nil
}

// path derives correct in-bucket path for a file given its type.
func (s *Service) path(fileType, fileEndName string) string {
	s.log.Debug().Msg("calling `path` method")
	switch fileType {
	case "internal_raw_data":
		return s.cfg.S3Storage.FolderInternal + "/" + fileEndName
	case "external_raw_data":
		return s.cfg.S3Storage.FolderExternal + "/" + fileEndName
	case "binary_raw_data":
		return s.cfg.S3Storage.FolderBinary + "/" + fileEndName
	default:
		s.log.Warn().Msg(fmt.Sprintf("invalid file type for upload: %s", fileType))
		return "temp/" + fileEndName
	}
}

// DownloadFile performs data download from S3.
func (s *Service) DownloadFile(fileName string) error {
	s.log.Debug().Msg("calling `DownloadFile` method")
	res, err := s.s3down.GetObjectWithContext(s.syncUtils.Ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.S3Storage.BucketUpload),
		Key:    aws.String(path.Join(s.cfg.S3Storage.FolderUpload, fileName)),
	})
	if err != nil {
		s.log.Error().Err(err).Msg(errors.FileDownloadError)
		return err
	}

	localFile, err := os.Create(s.cfg.Docker.MountDir + "/source/" + fileName)
	if err != nil {
		s.log.Error().Err(err).Msg(errors.FileOpeningError)
		return err
	}
	defer func(localFile *os.File) {
		err := localFile.Close()
		if err != nil {
			panic(err)
		}
	}(localFile)
	_, err = io.Copy(localFile, res.Body)
	if err != nil {
		s.log.Error().Err(err).Msg(errors.FileSavingError)
		return err
	}
	return nil
}
