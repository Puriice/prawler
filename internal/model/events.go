package model

import (
	"net/url"
	"time"

	"github.com/purrice/prawler/internal/enum/hosts"
	"github.com/purrice/prawler/internal/types"
)

var (
	ValidSeedEvent = []string{hosts.HostProduced, hosts.HostBlacklistAdd}
)

type Event struct {
	EventType *string `json:"event_type,omitempty"`
	Payload   any     `json:"payload,omitempty"`
}

func (s Event) IsValid() error {
	if s.EventType == nil {
		return types.ErrMissingField
	}

	return nil
}

type HostPayload struct {
	Host      *string    `json:"host,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

func (s HostPayload) IsValid() error {
	if s.Host == nil || s.Timestamp == nil {
		return types.ErrMissingField
	}

	if *s.Host == "" {
		return types.ErrInvalidSeed
	}

	return nil
}

func (s HostPayload) GetHost() (*url.URL, error) {
	url, err := url.Parse(*s.Host)

	if err != nil {
		return nil, types.ErrInvalidSeed
	}

	return url, nil
}
