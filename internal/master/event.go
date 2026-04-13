package master

import (
	"slices"

	"github.com/purrice/prawler/internal/types"
)

type Event struct {
	Type    EventType `json:"event_type"`
	Payload any       `json:"payload"`
}

func (e Event) IsValid() error {
	if !slices.Contains(ValidEventType, e.Type) {
		return types.ErrInvalidEventType
	}

	return nil
}
