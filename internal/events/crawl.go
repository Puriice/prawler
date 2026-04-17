package events

import (
	"github.com/purrice/prawler/internal/types"
)

type CrawlEventType string

const (
	CrawlURI CrawlEventType = "URI"
)

type CrawlEvent struct {
	Type    CrawlEventType `json:"event_type"`
	Payload URIPayload     `json:"payload"`
}

func (s CrawlEvent) IsValid() error {
	if s.Type == "" {
		return types.ErrMissingField
	}

	return nil
}
