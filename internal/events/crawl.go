package events

import (
	"github.com/google/uuid"
	"github.com/purrice/prawler/internal/types"
)

type CrawlEventType string

const (
	CrawlURI CrawlEventType = "URI"
)

type CrawlPayload struct {
	URIPayload
	PageUUID string `json:"page_uuid,omitempty"`
}

type CrawlEvent struct {
	Type    CrawlEventType `json:"event_type"`
	Payload CrawlPayload   `json:"payload"`
}

func (s CrawlEvent) IsValid() error {
	if s.Type == "" {
		return types.ErrMissingField
	}

	return nil
}

func (p CrawlPayload) IsValid() error {
	if err := p.URIPayload.IsValid(); err != nil {
		return err
	}

	if _, err := uuid.Parse(p.PageUUID); err != nil {
		return err
	}

	return nil
}
