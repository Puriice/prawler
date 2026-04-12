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

type URIPayload struct {
	URI       *string    `json:"host,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

func (s URIPayload) IsValid() error {
	if s.URI == nil || s.Timestamp == nil {
		return types.ErrMissingField
	}

	if *s.URI == "" {
		return types.ErrMissingURI
	}

	return nil
}

func (s URIPayload) GetHost() (*url.URL, error) {
	url, err := url.Parse(*s.URI)

	if err != nil {
		return nil, types.ErrMissingURI
	}

	return url, nil
}
