// Package errors provides string codes for error instantiation.

package errors

const (
	InvalidContentType      = "invalid content type"
	RequestBodyReadingError = "failed to read request body"
	UnmarshallingError      = "failed to unmarshall request body"
	MarshallingError        = "failed to marshall response body"
)
