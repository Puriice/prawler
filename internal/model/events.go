package model

import (
	"net/url"
	"slices"
	"time"

	"github.com/purrice/prawler/internal/enum/hosts"
)

var (
	ValidSeedEvent = []string{hosts.HostProduced, hosts.HostBlacklistAdd}
)

type HostEvent struct {
	EventType *string       `json:"event_type,omitempty"`
	Payload   *EventPayload `json:"payload,omitempty"`
}

func (s HostEvent) IsValid() error {
	if s.EventType == nil {
		return ErrMissingField
	}

	if !slices.Contains(ValidSeedEvent, *s.EventType) {
		return ErrInvalidEventType
	}

	return nil
}

type EventPayload struct {
	Host      *string    `json:"host,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

func (s EventPayload) IsValid() error {
	if s.Host == nil || s.Timestamp == nil {
		return ErrMissingField
	}

	if *s.Host == "" {
		return ErrInvalidSeed
	}

	return nil
}

func (s EventPayload) GetHost() (*url.URL, error) {
	url, err := url.Parse(*s.Host)

	if err != nil {
		return nil, ErrInvalidSeed
	}

	return url, nil
}
