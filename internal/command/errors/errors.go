// Package errors provides string codes for error instantiation.

package errors

const (
	FileReadingError             = "could not read data from file"
	TempFileOpeningError         = "could not open a temporary file"
	TempFileWritingError         = "could not dump uploaded multipart form data into a temporary file"
	ValidationRunError           = "could not run validation"
	ProcessingRunError           = "could not run processing"
	UserNotFoundError            = "could not find userID in DB"
	FileNotFoundError            = "could not find file name in DB"
	GettingProcessingStatusError = "could not find processing status in DB"
	GettingProductCodeError      = "could not find product code in DB"
)
