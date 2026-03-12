package model

import "errors"

var (
	ErrMissingField     = errors.New("Missing required field.")
	ErrInvalidField     = errors.New("Required field is invalid.")
	ErrInvalidEventType = errors.New("Unknown event types.")
	ErrInvalidSeed      = errors.New("Seed can't be empty.")
)
