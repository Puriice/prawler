package events

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/purrice/prawler/internal/types"
)

type FrontierEventType string

const (
	FrontierURIRegister  FrontierEventType = "prawler.frontier.uri"
	FrontierCrawlConfirm FrontierEventType = "prawler.frontier.confirm"
)

var ValidEventType = []FrontierEventType{
	FrontierURIRegister,
	FrontierCrawlConfirm,
}

type FrontierEvent struct {
	Type    FrontierEventType `json:"event_type"`
	Payload json.RawMessage   `json:"payload"`
}

func (e FrontierEvent) IsValid() error {
	if !slices.Contains(ValidEventType, e.Type) {
		return types.ErrInvalidEventType
	}

	return nil
}

type ConfirmPayload struct {
	PageUUID *string `json:"page_uuid,omitempty"`

	URI       *string `json:"uri,omitempty"`
	FinalURI  *string `json:"final_uri,omitempty"`
	Canonical *string `json:"canonical,omitempty"`

	Depth int `json:"depth,omitempty"`

	Timestamp time.Time `json:"timestamp,omitempty"`
}

func (p ConfirmPayload) IsValid() error {
	if p.PageUUID == nil || p.URI == nil {
		return types.ErrMissingURI
	}

	if *p.PageUUID == "" || *p.URI == "" {
		return types.ErrInvalidField
	}

	if _, err := uuid.Parse(*p.PageUUID); err != nil {
		return types.ErrInvalidUUID
	}

	if p.Canonical != nil && *p.Canonical == "" {
		return types.ErrInvalidField
	}

	if p.FinalURI != nil && *p.FinalURI == "" {
		return types.ErrInvalidField
	}

	return nil
}
