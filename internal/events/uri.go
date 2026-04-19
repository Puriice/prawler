package events

import (
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/purrice/prawler/internal/types"
)

type URIPayload struct {
	URI       *string    `json:"uri,omitempty"`
	Depth     int        `json:"depth,omitempty"`
	Source    *Source    `json:"source,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

type Source struct {
	PageUUID   *string `json:"page_uuid,omitempty"`
	AnchorText string  `json:"anchor_text"`
}

func (s Source) IsValid() error {
	if s.PageUUID == nil {
		return types.ErrMissingField
	}

	if _, err := uuid.Parse(*s.PageUUID); err != nil {
		return types.ErrInvalidUUID
	}

	return nil
}

func (s URIPayload) IsValid() error {
	if s.URI == nil || s.Timestamp == nil {
		return types.ErrMissingField
	}

	if *s.URI == "" {
		return types.ErrMissingURI
	}

	if s.Depth < 0 {
		return types.ErrInvalidField
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
