package frontier

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/purrice/prawler/internal/types"
)

type Event struct {
	Type    EventType       `json:"event_type"`
	Payload json.RawMessage `json:"payload"`
}

func (e Event) IsValid() error {
	if !slices.Contains(ValidEventType, e.Type) {
		return types.ErrInvalidEventType
	}

	return nil
}

type ConfirmPayload struct {
	URI       *string `json:"uri,omitempty"`
	FinalURI  *string `json:"final_uri,omitempty"`
	Canonical *string `json:"canonical,omitempty"`

	Depth int `json:"depth,omitempty"`

	Timestamp time.Time `json:"timestamp,omitempty"`
}

func (p ConfirmPayload) IsValid() error {
	if p.URI == nil {
		return types.ErrMissingURI
	}

	if *p.URI == "" {
		return types.ErrInvalidField
	}

	if p.Canonical != nil && *p.Canonical == "" {
		return types.ErrInvalidField
	}

	if p.FinalURI != nil && *p.FinalURI == "" {
		return types.ErrInvalidField
	}

	return nil
}
