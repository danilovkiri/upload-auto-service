// Package models provides data types and models used in package processor.

package models

type ValidationData struct {
	Mode   string `json:"mode"`
	Sex    string `json:"sex"`
	Err    string `json:"error"`
	Passed bool   `json:"passed"`
}
