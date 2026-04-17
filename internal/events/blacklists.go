package events

import (
	"slices"

	"github.com/purrice/prawler/internal/types"
)

type BlacklistEventType string

const (
	BlacklistAdd    BlacklistEventType = "blacklists.add"
	BlacklistRemove BlacklistEventType = "blacklists.remove"
)

var (
	ValidBlacklistEvents = []BlacklistEventType{BlacklistAdd, BlacklistRemove}
)

type BlacklistEvent struct {
	Type    BlacklistEventType `json:"event_type"`
	Payload URIPayload         `json:"payload"`
}

func (s BlacklistEvent) IsValid() error {
	if s.Type == "" {
		return types.ErrMissingField
	}

	if !slices.Contains(ValidBlacklistEvents, s.Type) {
		return types.ErrInvalidEventType
	}

	return nil
}
