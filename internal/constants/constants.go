// Package constants provides constants.

package constants

const (
	ValidationStatusNew     = "new"
	ValidationStatusRunning = "running"
	ValidationStatusValid   = "valid"
	ValidationStatusInvalid = "invalid"
	ValidationStatusError   = "error"

	ProcessingStatusNew     = "new"
	ProcessingStatusRunning = "running"
	ProcessingStatusDone    = "done"
	ProcessingStatusError   = "error"

	NA = "NA"
)

var ValidValidationStatuses = []string{
	ValidationStatusNew,
	ValidationStatusRunning,
	ValidationStatusValid,
	ValidationStatusInvalid,
	ValidationStatusError}

var ValidProcessingStatuses = []string{
	ProcessingStatusNew,
	ProcessingStatusRunning,
	ProcessingStatusDone,
	ProcessingStatusError}
