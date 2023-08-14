// Package processor provides functionality for running docker image and catching its output.

package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/constants"
	"upload-service-auto/internal/processor/errors"
	"upload-service-auto/internal/processor/v1/models"
	"upload-service-auto/internal/s3/s3"
	"upload-service-auto/internal/storage/v1/psql"
	"upload-service-auto/internal/syncutils"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

// Processor defines an object and sets its attributes.
type Processor struct {
	st        *psql.Storage
	cfg       *config.Config
	log       *zerolog.Logger
	s3        *s3.Service
	syncUtils *syncutils.SyncUtils
}

// NewProcessor initializes a new Processor instance.
func NewProcessor(storage *psql.Storage, config *config.Config, logger *zerolog.Logger, s3 *s3.Service, syncUtils *syncutils.SyncUtils) *Processor {
	logger.Debug().Msg("calling initializer of processor service")
	return &Processor{
		st:        storage,
		cfg:       config,
		log:       logger,
		s3:        s3,
		syncUtils: syncUtils,
	}
}

// prepareCommand prepares shell comand for execution.
func (p *Processor) prepareCommand(executable string, cliArgs []string, catcher io.Writer) *exec.Cmd {
	p.log.Debug().Msg("calling `prepareCommand` method")
	cmdGo := &exec.Cmd{
		Path:   executable,
		Args:   cliArgs,
		Stdout: catcher,
		Stderr: os.Stderr,
	}
	return cmdGo
}

// RunValidation runs validation command and interacts with DB.
func (p *Processor) RunValidation(ctx context.Context, fileName string, dryRun, fromQueue bool) (*models.ValidationData, error) {
	p.log.Debug().Msg("calling `RunValidation` method")
	if fromQueue {
		err := p.s3.DownloadFile(fileName)
		if err != nil {
			p.log.Error().Err(err).Msg(errors.DownloadS3Error)
			return nil, err
		}
	}

	executable := p.cfg.Docker.DockerExecutable
	args := []string{
		executable,
		"run",
		"--rm",
		"-v",
		fmt.Sprintf("%s:/mnt", p.cfg.Docker.MountDir),
		p.cfg.Docker.DockerImageName,
		"main.py",
		"validate",
		"--input",
		fileName,
	}

	if !dryRun {
		err := p.st.UpdateValidationStatus(ctx, fileName, constants.ValidationStatusRunning)
		if err != nil {
			p.log.Error().Err(err).Msg(errors.ValidationStatusUpdateError)
			return nil, err
		}
	}
	catcher := &bytes.Buffer{}
	cmd := p.prepareCommand(executable, args, catcher)
	p.log.Info().Msg(cmd.String())
	err := cmd.Run()
	if err != nil {
		p.log.Error().Err(err).Msg(errors.ValidationSubprocessError)
		if !dryRun {
			err := p.st.UpdateValidationStatus(ctx, fileName, constants.ValidationStatusError)
			if err != nil {
				p.log.Error().Err(err).Msg(errors.ValidationStatusUpdateError)
				return nil, err
			}
		}
		return nil, err
	}

	cmdStdout := catcher.Bytes()
	p.log.Info().Str("data", string(cmdStdout)).Msg("validation completed")

	var cmdOutput *models.ValidationData
	err = json.Unmarshal(cmdStdout, &cmdOutput)
	if err != nil {
		p.log.Error().Err(err).Str("data", string(cmdStdout)).Msg(errors.ValidationDataUnmarshalError)
		return nil, err
	}

	if !dryRun {
		if cmdOutput.Passed {
			err = p.st.UpdateValidationStatus(ctx, fileName, constants.ValidationStatusValid)
			if err != nil {
				p.log.Error().Err(err).Msg(errors.ValidationStatusUpdateError)
				return nil, err
			}
		} else {
			err = p.st.UpdateValidationStatus(ctx, fileName, constants.ValidationStatusInvalid)
			if err != nil {
				p.log.Error().Err(err).Msg(errors.ValidationStatusUpdateError)
				return nil, err
			}
		}
	}
	return cmdOutput, nil
}

// RunProcessing runs processing command and interacts with DB.
func (p *Processor) RunProcessing(ctx context.Context, fileName, barcode string, dryRun, fromQueue bool) error {
	p.log.Debug().Msg("calling `RunProcessing` method")
	if fromQueue {
		err := p.s3.DownloadFile(fileName)
		if err != nil {
			p.log.Error().Err(err).Msg(errors.DownloadS3Error)
			return err
		}
	}

	executable := p.cfg.Docker.DockerExecutable
	args := []string{
		executable,
		"run",
		"--rm",
		"-v",
		fmt.Sprintf("%s:/mnt", p.cfg.Docker.MountDir),
		p.cfg.Docker.DockerImageName,
		"main.py",
		"process",
		"--input",
		fileName,
		"--barcode",
		barcode,
	}

	err := p.st.UpdateProcessingStatus(ctx, fileName, constants.ProcessingStatusRunning)
	if err != nil {
		p.log.Error().Err(err).Msg(errors.ProcessingStatusUpdateError)
		return err
	}

	cmd := p.prepareCommand(executable, args, os.Stdout)

	err = cmd.Run()
	if err != nil {
		p.log.Error().Err(err).Msg(errors.ProcessingSubprocessError)
		err := p.st.UpdateProcessingStatus(ctx, fileName, constants.ProcessingStatusError)
		if err != nil {
			p.log.Error().Err(err).Msg(errors.ProcessingStatusUpdateError)
			return err
		}
		return err
	}
	err = p.st.UpdateProcessingStatus(ctx, fileName, constants.ProcessingStatusDone)
	if err != nil {
		p.log.Error().Err(err).Msg(errors.ProcessingStatusUpdateError)
		return err
	}
	if !dryRun {
		err = p.uploadData(barcode)
		if err != nil {
			p.log.Error().Err(err).Msg(errors.UploadRoutineError)
			return err
		}
	}
	return nil
}

// uploadData uploads data to S3.
func (p *Processor) uploadData(barcode string) error {
	p.log.Debug().Msg("calling `uploadData` method")
	files := map[string][]string{
		fmt.Sprintf("%s/raw_data/atlas_raw_data/%s.txt", p.cfg.Docker.MountDir, barcode):    {"internal_raw_data", fmt.Sprintf("%s.txt", barcode)},
		fmt.Sprintf("%s/raw_data/external_raw_data/%s.txt", p.cfg.Docker.MountDir, barcode): {"external_raw_data", fmt.Sprintf("%s.txt", barcode)},
		fmt.Sprintf("%s/raw_data/binary/%s.bed", p.cfg.Docker.MountDir, barcode):            {"binary_raw_data", fmt.Sprintf("%s.bed", barcode)},
		fmt.Sprintf("%s/raw_data/binary/%s.bim", p.cfg.Docker.MountDir, barcode):            {"binary_raw_data", fmt.Sprintf("%s.bim", barcode)},
		fmt.Sprintf("%s/raw_data/binary/%s.fam", p.cfg.Docker.MountDir, barcode):            {"binary_raw_data", fmt.Sprintf("%s.fam", barcode)},
	}

	g := &errgroup.Group{}
	for path, meta := range files {
		filePath := path
		fileType := meta[0]
		fileEndName := meta[1]
		g.Go(func() error {
			return p.s3.UploadFile(filePath, fileType, fileEndName)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
