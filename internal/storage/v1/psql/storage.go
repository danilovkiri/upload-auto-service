// Package psql provides PSQL storage service.

package psql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
	"upload-service-auto/internal/config"
	"upload-service-auto/internal/constants"
	storageErrors "upload-service-auto/internal/storage/errors"
	"upload-service-auto/internal/syncutils"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog"
)

// Storage defines a new object and sets its attributes.
type Storage struct {
	mu        sync.Mutex
	cfg       *config.Config
	DB        *sql.DB
	log       *zerolog.Logger
	syncUtils *syncutils.SyncUtils
}

// checkInSlice checks that a string is contained within a slice.
func (s *Storage) checkInSlice(slice []string, value string) bool {
	s.log.Debug().Msg("calling `checkInSlice` method")
	for _, x := range slice {
		if x == value {
			return true
		}
	}
	return false
}

// DropAll drops the DB tables.
func (s *Storage) DropAll() error {
	s.log.Debug().Msg("calling `DropAll` method")
	ctx, cancel := context.WithTimeout(s.syncUtils.Ctx, 1000*time.Millisecond)
	defer cancel()
	defer s.syncUtils.SyncCancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	var queries []string
	query := `DROP TABLE IF  EXISTS users;`
	queries = append(queries, query)

	query = `DROP TABLE IF EXISTS files ;`
	queries = append(queries, query)

	query = `DROP TABLE IF EXISTS products;`
	queries = append(queries, query)

	query = `DROP TABLE IF EXISTS validation;`
	queries = append(queries, query)

	query = `DROP TABLE IF EXISTS processing;`
	queries = append(queries, query)

	for _, subquery := range queries {
		_, err := s.DB.ExecContext(ctx, subquery)
		if err != nil {
			return err
		}
	}
	return nil
}

// Migrate creates the DB tables.
func (s *Storage) Migrate() error {
	s.log.Debug().Msg("calling `Migrate` method")
	ctx, cancel := context.WithTimeout(s.syncUtils.Ctx, 1000*time.Millisecond)
	defer cancel()
	defer s.syncUtils.SyncCancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	var queries []string
	query := `CREATE TABLE IF NOT EXISTS users (
		id            BIGSERIAL   NOT NULL UNIQUE,
		user_id       TEXT        NOT NULL UNIQUE,
		created_at TIMESTAMPTZ NOT NULL  
	);`
	queries = append(queries, query)

	query = `CREATE TABLE IF NOT EXISTS files (
		id           BIGSERIAL      NOT NULL UNIQUE,
		user_id      TEXT           NOT NULL UNIQUE,
		file_name    TEXT           NOT NULL UNIQUE,
		updated_at   TIMESTAMPTZ    NOT NULL  
	);`
	queries = append(queries, query)

	query = `CREATE TABLE IF NOT EXISTS products (
		id            BIGSERIAL  NOT NULL UNIQUE,
		user_id       TEXT       NOT NULL UNIQUE,
		product_code  TEXT       NOT NULL
	);`
	queries = append(queries, query)

	query = `CREATE TABLE IF NOT EXISTS validation (
		id           BIGSERIAL   NOT NULL UNIQUE,
		file_name    TEXT        NOT NULL UNIQUE,
		status       TEXT        NOT NULL,
		updated_at   TIMESTAMPTZ NOT NULL  
	);`
	queries = append(queries, query)

	query = `CREATE TABLE IF NOT EXISTS processing (
		id           BIGSERIAL   NOT NULL UNIQUE,
		file_name    TEXT        NOT NULL UNIQUE,
		barcode      TEXT,
		status       TEXT        NOT NULL,
		updated_at   TIMESTAMPTZ NOT NULL
	);`
	queries = append(queries, query)

	for _, subquery := range queries {
		_, err := s.DB.ExecContext(ctx, subquery)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewStorage initializes a new Storage instance.
func NewStorage(cfg *config.Config, logger *zerolog.Logger, syncUtils *syncutils.SyncUtils) *Storage {
	logger.Debug().Msg("calling initializer of storage service")
	db, err := sql.Open("pgx", cfg.DB.DatabaseDSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not open a DB connection")
	}
	st := Storage{
		cfg:       cfg,
		DB:        db,
		log:       logger,
		syncUtils: syncUtils,
	}
	logger.Debug().Msg("DB connection was established")

	st.syncUtils.Wg.Add(1)
	go func() {
		defer st.syncUtils.Wg.Done()
		<-st.syncUtils.Ctx.Done()
		err = st.DB.Close()
		if err != nil {
			logger.Fatal().Err(err).Msg("could not close DB connection")
		}
		logger.Debug().Msg("PSQL DB connection was closed")
	}()

	return &st
}

// RemoveUserData removes all data for one user.
func (s *Storage) RemoveUserData(ctx context.Context, userID, fileName string) error {
	s.log.Debug().Msg("calling `RemoveUserData` method")
	newDeleteStmtUsers, err := s.DB.PrepareContext(ctx, "DELETE FROM users WHERE user_id = $1")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newDeleteStmtUsers.Close()
	newDeleteStmtFiles, err := s.DB.PrepareContext(ctx, "DELETE FROM files WHERE user_id = $1")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newDeleteStmtFiles.Close()
	newDeleteStmtProducts, err := s.DB.PrepareContext(ctx, "DELETE FROM products WHERE user_id = $1")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newDeleteStmtProducts.Close()
	newDeleteStmtValidation, err := s.DB.PrepareContext(ctx, "DELETE FROM validation WHERE file_name = $1")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newDeleteStmtValidation.Close()
	newDeleteStmtProcessing, err := s.DB.PrepareContext(ctx, "DELETE FROM processing WHERE file_name = $1")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newDeleteStmtProcessing.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)

	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := newDeleteStmtUsers.ExecContext(ctx, userID)
		if err != nil {
			chanEr <- err
			return
		}

		_, err = newDeleteStmtFiles.ExecContext(ctx, userID)
		if err != nil {
			chanEr <- err
			return
		}

		_, err = newDeleteStmtProducts.ExecContext(ctx, userID)
		if err != nil {
			chanEr <- err
			return
		}

		_, err = newDeleteStmtValidation.ExecContext(ctx, fileName)
		if err != nil {
			chanEr <- err
			return
		}

		_, err = newDeleteStmtProcessing.ExecContext(ctx, fileName)
		if err != nil {
			chanEr <- err
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("userID", userID).Msg("removing user failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("userID", userID).Msg("removing user failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("userID", userID).Msg("removing user done")
		return nil
	}
}

// GetAllUserIDs retrieves all user identifiers currently stored in DB.
func (s *Storage) GetAllUserIDs(ctx context.Context) ([]string, error) {
	s.log.Debug().Msg("calling `GetAllUserIDs` method")
	getUsersStmt, err := s.DB.PrepareContext(ctx, "SELECT user_id from users")
	if err != nil {
		s.log.Error().Err(err).Msg("could not prepare statement")
		return nil, &storageErrors.StatementPSQLError{Err: err}
	}
	defer getUsersStmt.Close()

	chanOk := make(chan []string)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		rows, err := getUsersStmt.QueryContext(ctx)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		defer rows.Close()

		var queryOutput []string
		for rows.Next() {
			var queryOutputRow string
			err = rows.Scan(&queryOutputRow)
			if err != nil {
				chanEr <- &storageErrors.ScanningPSQLError{Err: err}
				return
			}
			queryOutput = append(queryOutput, queryOutputRow)
		}
		err = rows.Err()
		if err != nil {
			chanEr <- &storageErrors.ScanningPSQLError{Err: err}
		}
		chanOk <- queryOutput
	}()
	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Msg("getting all users failed")
		return nil, &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Msg("getting all users failed")
		return nil, methodErr
	case result := <-chanOk:
		s.log.Info().Msg("getting all users done")
		return result, nil
	}
}

// AddNewUserID adds a new user to DB.
func (s *Storage) AddNewUserID(ctx context.Context, userID string) error {
	s.log.Debug().Msg("calling `AddNewUserID` method")
	newUserStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO users (user_id, created_at) VALUES ($1, $2)")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newUserStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := newUserStmt.ExecContext(ctx, userID, time.Now().Format(time.RFC3339))
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: userID}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("userID", userID).Msg("adding new user failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("userID", userID).Msg("adding new user failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("userID", userID).Msg("adding new user done")
		return nil
	}
}

// CheckUserID checks that a user is stored in DB.
func (s *Storage) CheckUserID(ctx context.Context, userID string) error {
	s.log.Debug().Msg("calling `CheckUserID` method")
	checkUserStmt, err := s.DB.PrepareContext(ctx, "SELECT COUNT(1) > 0 from users where user_id = $1")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer checkUserStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		var userIsPresent bool
		err := checkUserStmt.QueryRowContext(ctx, userID).Scan(&userIsPresent)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		if !userIsPresent {
			chanEr <- &storageErrors.NotFoundError{Err: errors.New("user was not found")}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("userID", userID).Msg("checking user failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("userID", userID).Msg("checking user failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("userID", userID).Msg("checking user done")
		return nil
	}
}

// AddNewUserFilePair creates a new user-file entry.
func (s *Storage) AddNewUserFilePair(ctx context.Context, userID, fileName string) error {
	s.log.Debug().Msg("calling `AddNewUserFilePair` method")
	newFileStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO files (user_id, file_name, updated_at) VALUES ($1, $2, $3)")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newFileStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := newFileStmt.ExecContext(ctx, userID, fileName, time.Now().Format(time.RFC3339))
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: userID}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("userID", userID).Msg("adding new file failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("userID", userID).Msg("adding new file failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("userID", userID).Msg("adding new file done")
		return nil
	}
}

// UpdateUserFilePair updates a user-file pair with a new file.
func (s *Storage) UpdateUserFilePair(ctx context.Context, userID, fileName string) error {
	s.log.Debug().Msg("calling `UpdateUserFilePair` method")
	updateFileStmt, err := s.DB.PrepareContext(ctx, "UPDATE files SET (file_name, updated_at) = ($1, $2) WHERE user_id = $3")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer updateFileStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := updateFileStmt.ExecContext(ctx, fileName, time.Now().Format(time.RFC3339), userID)
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: userID}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("userID", userID).Msg("updating new file failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("userID", userID).Msg("updating new file failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("userID", userID).Msg("updating new file done")
		return nil
	}
}

// GetFileNameForUser retrieves an active filename for a user.
func (s *Storage) GetFileNameForUser(ctx context.Context, userID string) (string, error) {
	s.log.Debug().Msg("calling `GetFileNameForUser` method")
	getFileStmt, err := s.DB.PrepareContext(ctx, "SELECT file_name FROM files WHERE user_id = $1")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return "", &storageErrors.StatementPSQLError{Err: err}
	}
	defer getFileStmt.Close()
	chanOk := make(chan string)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		var fileName string
		err := getFileStmt.QueryRowContext(ctx, userID).Scan(&fileName)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- fileName
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("userID", userID).Msg("getting filename failed")
		return "", &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("userID", userID).Msg("getting filename file failed")
		return "", methodErr
	case result := <-chanOk:
		s.log.Info().Str("userID", userID).Msg("getting filename file done")
		return result, nil
	}
}

// AddNewValidationEntry adds new validation data.
func (s *Storage) AddNewValidationEntry(ctx context.Context, fileName string) error {
	s.log.Debug().Msg("calling `AddNewValidationEntry` method")
	newEntryStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO validation (file_name, status, updated_at) VALUES ($1, $2, $3)")
	if err != nil {
		s.log.Error().Err(err).Str("fileName", fileName).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newEntryStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := newEntryStmt.ExecContext(ctx, fileName, constants.ValidationStatusNew, time.Now().Format(time.RFC3339))
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: fileName}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("fileName", fileName).Msg("adding validation entry failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("fileName", fileName).Msg("adding validation entry failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("fileName", fileName).Msg("adding validation entry done")
		return nil
	}
}

// UpdateValidationStatus updates validation status for a file.
func (s *Storage) UpdateValidationStatus(ctx context.Context, fileName, status string) error {
	s.log.Debug().Msg("calling `UpdateValidationStatus` method")
	if !s.checkInSlice(constants.ValidValidationStatuses, status) {
		err := errors.New("invalid status")
		s.log.Error().Err(err).Str("fileName", fileName).Msg(fmt.Sprintf("status %s is invalid", status))
		return err
	}

	updateValidityStmt, err := s.DB.PrepareContext(ctx, "UPDATE validation SET (status, updated_at) = ($1, $2) WHERE file_name = $3")
	if err != nil {
		s.log.Error().Err(err).Str("fileName", fileName).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer updateValidityStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := updateValidityStmt.ExecContext(ctx, status, time.Now().Format(time.RFC3339), fileName)
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: fileName}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("fileName", fileName).Msg("updating validity failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("fileName", fileName).Msg("updating validity failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("fileName", fileName).Msg("updating validity done")
		return nil
	}
}

// CheckIsValid checks that validation is completed and the file is valid for further processing.
func (s *Storage) CheckIsValid(ctx context.Context, fileName string) error {
	s.log.Debug().Msg("calling `CheckIsValid` method")
	checkValidityStmt, err := s.DB.PrepareContext(ctx, "SELECT COUNT(1) > 0 from validation where file_name = $1 AND status = $2")
	if err != nil {
		s.log.Error().Err(err).Str("fileName", fileName).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer checkValidityStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		var fileIsValid bool
		err := checkValidityStmt.QueryRowContext(ctx, fileName, constants.ValidationStatusValid).Scan(&fileIsValid)
		if err != nil {
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		if !fileIsValid {
			chanEr <- &storageErrors.NotFoundError{Err: errors.New("file is not valid")}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("fileName", fileName).Msg("checking validity failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("fileName", fileName).Msg("checking validity failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("fileName", fileName).Msg("checking validity done")
		return nil
	}
}

// AddNewProcessingEntry adds new processing entry to DB.
func (s *Storage) AddNewProcessingEntry(ctx context.Context, fileName, barcode string) error {
	s.log.Debug().Msg("calling `AddNewProcessingEntry` method")
	newEntryStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO processing (file_name, barcode, status, updated_at) VALUES ($1, $2, $3, $4)")
	if err != nil {
		s.log.Error().Err(err).Str("fileName", fileName).Str("barcode", barcode).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newEntryStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := newEntryStmt.ExecContext(ctx, fileName, barcode, constants.ProcessingStatusNew, time.Now().Format(time.RFC3339))
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: fileName}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("fileName", fileName).Str("barcode", barcode).Msg("adding processing entry failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("fileName", fileName).Str("barcode", barcode).Msg("adding processing entry failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("fileName", fileName).Str("barcode", barcode).Msg("adding processing entry done")
		return nil
	}
}

// UpdateProcessingStatus updates processing status of a file.
func (s *Storage) UpdateProcessingStatus(ctx context.Context, fileName, status string) error {
	s.log.Debug().Msg("calling `UpdateProcessingStatus` method")
	if !s.checkInSlice(constants.ValidProcessingStatuses, status) {
		err := errors.New("invalid status")
		s.log.Error().Err(err).Str("fileName", fileName).Msg(fmt.Sprintf("status %s is invalid", status))
		return err
	}

	updateProcessingStmt, err := s.DB.PrepareContext(ctx, "UPDATE processing SET (status, updated_at) = ($1, $2) WHERE file_name = $3")
	if err != nil {
		s.log.Error().Err(err).Str("fileName", fileName).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer updateProcessingStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := updateProcessingStmt.ExecContext(ctx, status, time.Now().Format(time.RFC3339), fileName)
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: fileName}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("fileName", fileName).Msg("updating processing failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("fileName", fileName).Msg("updating processing failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("fileName", fileName).Msg("updating processing done")
		return nil
	}
}

// GetProcessingStatus retrieves processing status for a file.
func (s *Storage) GetProcessingStatus(ctx context.Context, fileName string) (string, error) {
	s.log.Debug().Msg("calling `GetProcessingStatus` method")
	checkProcessingStmt, err := s.DB.PrepareContext(ctx, "SELECT status from processing where file_name = $1")
	if err != nil {
		s.log.Error().Err(err).Str("fileName", fileName).Msg("could not prepare statement")
		return "", &storageErrors.StatementPSQLError{Err: err}
	}
	defer checkProcessingStmt.Close()
	chanOk := make(chan string)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		var processingStatus string
		err := checkProcessingStmt.QueryRowContext(ctx, fileName).Scan(&processingStatus)
		if err != nil {
			if err == sql.ErrNoRows {
				chanOk <- constants.NA
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- processingStatus
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("fileName", fileName).Msg("checking processing failed")
		return "", &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("fileName", fileName).Msg("checking processing failed")
		return "", methodErr
	case result := <-chanOk:
		s.log.Info().Str("fileName", fileName).Msg("checking processing done")
		return result, nil
	}
}

// AddNewProductCode adds a new product code for a user to DB.
func (s *Storage) AddNewProductCode(ctx context.Context, userID, productCode string) error {
	s.log.Debug().Msg("calling `AddNewProductCode` method")
	newProdCodeStmt, err := s.DB.PrepareContext(ctx, "INSERT INTO products (user_id, product_code) VALUES ($1, $2)")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Str("productCode", productCode).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer newProdCodeStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := newProdCodeStmt.ExecContext(ctx, userID, productCode)
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: userID}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("userID", userID).Str("productCode", productCode).Msg("adding product code failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("userID", userID).Str("productCode", productCode).Msg("adding product code failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("userID", userID).Str("productCode", productCode).Msg("adding product code done")
		return nil
	}
}

// UpdateProductCode updates a product code for a user.
func (s *Storage) UpdateProductCode(ctx context.Context, userID, productCode string) error {
	s.log.Debug().Msg("calling `UpdateProductCode` method")
	updProdCodeStmt, err := s.DB.PrepareContext(ctx, "UPDATE products set product_code = $1 where user_id = $2")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Str("productCode", productCode).Msg("could not prepare statement")
		return &storageErrors.StatementPSQLError{Err: err}
	}
	defer updProdCodeStmt.Close()
	chanOk := make(chan bool)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := updProdCodeStmt.ExecContext(ctx, productCode, userID)
		if err != nil {
			if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
				chanEr <- &storageErrors.AlreadyExistsError{Err: err, ID: userID}
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- true
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("userID", userID).Str("productCode", productCode).Msg("adding product code failed")
		return &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("userID", userID).Str("productCode", productCode).Msg("adding product code failed")
		return methodErr
	case <-chanOk:
		s.log.Info().Str("userID", userID).Str("productCode", productCode).Msg("adding product code done")
		return nil
	}
}

// GetProductCode retrieves a product code for a user.
func (s *Storage) GetProductCode(ctx context.Context, userID string) (string, error) {
	s.log.Debug().Msg("calling `GetProductCode` method")
	getProdCodeStmt, err := s.DB.PrepareContext(ctx, "SELECT product_code from products where user_id = $1")
	if err != nil {
		s.log.Error().Err(err).Str("userID", userID).Msg("could not prepare statement")
		return "", &storageErrors.StatementPSQLError{Err: err}
	}
	defer getProdCodeStmt.Close()
	chanOk := make(chan string)
	chanEr := make(chan error)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		var productCode string
		err := getProdCodeStmt.QueryRowContext(ctx, userID).Scan(&productCode)
		if err != nil {
			if err == sql.ErrNoRows {
				chanOk <- constants.NA
				return
			}
			chanEr <- &storageErrors.ExecutionPSQLError{Err: err}
			return
		}
		chanOk <- productCode
	}()

	select {
	case <-ctx.Done():
		s.log.Error().Err(ctx.Err()).Str("userID", userID).Msg("getting product code failed")
		return "", &storageErrors.ContextTimeoutExceededError{Err: ctx.Err()}
	case methodErr := <-chanEr:
		s.log.Error().Err(methodErr).Str("userID", userID).Msg("getting product code failed")
		return "", methodErr
	case result := <-chanOk:
		s.log.Info().Str("userID", userID).Msg("getting product code done")
		return result, nil
	}
}
