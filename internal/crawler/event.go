package crawler

import (
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/types"
)

type EventType string

const (
	EventURI EventType = "URI"
)

type Event struct {
	Type    EventType        `json:"event_type"`
	Payload model.URIPayload `json:"payload"`
}

func (s Event) IsValid() error {
	if s.Type == "" {
		return types.ErrMissingField
	}

	return nil
}
