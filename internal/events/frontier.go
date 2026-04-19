package events

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/purrice/prawler/internal/enum"
	"github.com/purrice/prawler/internal/types"
)

type FrontierEventType string

const (
	FrontierURIRegister  FrontierEventType = "prawler.frontier.uri"
	FrontierCrawlConfirm FrontierEventType = "prawler.frontier.confirm"
	FrontierBackoff      FrontierEventType = "prawler.frontier.backoff"
)

var ValidEventType = []FrontierEventType{
	FrontierURIRegister,
	FrontierCrawlConfirm,
	FrontierBackoff,
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
	Status     enum.PageStatus `json:"status,omitempty"`
	HTTPStatus int             `json:"http_status,omitempty"`

	PageUUID *string `json:"page_uuid,omitempty"`

	URI       *string `json:"uri,omitempty"`
	FinalURI  string  `json:"final_uri,omitempty"`
	Canonical string  `json:"canonical,omitempty"`

	Depth int `json:"depth,omitempty"`

	Timestamp time.Time `json:"timestamp"`
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

	if !p.Status.IsValid() {
		return types.ErrInvalidField
	}

	if p.HTTPStatus < 100 || p.HTTPStatus > 599 {
		return types.ErrInvalidField
	}

	return nil
}

type BackoffPayload struct {
	PageUUID *string `json:"page_uuid,omitempty"`
	Depth    int     `json:"depth"`

	URI        *string       `json:"url,omitempty"`
	HTTPStatus int           `json:"http_status,omitempty"`
	RetryAfter time.Duration `json:"retry_after"`

	Timestamp time.Time `json:"timestamp"`
}

func (p BackoffPayload) IsValid() error {
	if p.PageUUID == nil || p.URI == nil {
		return types.ErrMissingField
	}

	if p.HTTPStatus < 100 || p.HTTPStatus > 599 {
		return types.ErrInvalidField
	}

	if _, err := uuid.Parse(*p.PageUUID); err != nil {
		return types.ErrInvalidUUID
	}

	return nil
}
