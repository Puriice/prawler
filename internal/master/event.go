package master

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/purrice/prawler/internal/types"
)

type Event struct {
	Type    EventType `json:"event_type"`
	Payload *Payload  `json:"payload"`
}

type Payload struct {
	UUID      *string    `json:"uuid"`
	Timestamp *time.Time `json:"timestamp"`
}

func (e Event) IsValid() error {
	if !slices.Contains(ValidEventType, e.Type) {
		return types.ErrInvalidEventType
	}

	return e.Payload.IsValid()
}

func (p Payload) IsValid() error {
	if p.UUID == nil || p.Timestamp == nil {
		return types.ErrInvalidField
	}

	if err := uuid.Validate(*p.UUID); err != nil {
		return types.ErrInvalidUUID
	}

	if p.Timestamp.After(time.Now()) {
		return types.ErrInvalidTimestamp
	}

	return nil
}
