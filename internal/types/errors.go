package types

import "errors"

var (
	ErrUnproccessableInput = errors.New("Invalid Input type")
	ErrMissingField        = errors.New("Missing required field.")
	ErrInvalidField        = errors.New("Required field is invalid.")
	ErrInvalidUUID         = errors.New("Invalid UUID")
	ErrInvalidTimestamp    = errors.New("Invalid timestamp. Maybe it was in the future?")
	ErrInvalidEventType    = errors.New("Unknown event types.")
	ErrMissingURI          = errors.New("URI can't be empty.")
	ErrInvalidPaylod       = errors.New("Payload is in invalid structure")
)
