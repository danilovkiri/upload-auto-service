// Package errors provides string codes for error instantiation.

package errors

const (
	ValidationRunError           = "could not run validation"
	ProcessingRunError           = "could not run processing"
	AddingUserError              = "could not add user to DB"
	UserNotFoundError            = "could not find userID in DB"
	FileNotFoundError            = "could not find file name in DB"
	InvalidFileError             = "file is not valid"
	GettingProcessingStatusError = "could not find processing status in DB"
	GettingProductCodeError      = "could not find product code in DB"
	AddingFileError              = "could not add a new file name"
	AddingValidationEntryError   = "could not add a new validation entry"
	AddingProcessingEntryError   = "could not add a new processing entry"
	AddingProductCodeError       = "could not add a new product code"
	ProcessingInProgressError    = "processing is currently running and locked"
)
