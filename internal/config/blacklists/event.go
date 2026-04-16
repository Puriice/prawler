package blacklists

import (
	"slices"

	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/types"
)

type EventType string

const (
	EventBlacklistAdd    EventType = "blacklists.add"
	EventBlacklistRemove EventType = "blacklists.remove"
)

var (
	ValidBlacklistEvents = []EventType{EventBlacklistAdd, EventBlacklistRemove}
)

type Event struct {
	Type    EventType        `json:"event_type"`
	Payload model.URIPayload `json:"payload"`
}

func (s Event) IsValid() error {
	if s.Type == "" {
		return types.ErrMissingField
	}

	if !slices.Contains(ValidBlacklistEvents, s.Type) {
		return types.ErrInvalidEventType
	}

	return nil
}
