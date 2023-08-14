// Package errors provides string codes for error instantiation.

package errors

const (
	ValidationStatusUpdateError  = "could not update validation status"
	ValidationSubprocessError    = "could not run validation shell command"
	ValidationDataUnmarshalError = "could not unmarshall validation data"
	ProcessingStatusUpdateError  = "could not update processing status"
	ProcessingSubprocessError    = "could not run processing shell command"
	UploadRoutineError           = "could not execute S3 upload in a goroutine"
	DownloadS3Error              = "could not download file from S3"
)
