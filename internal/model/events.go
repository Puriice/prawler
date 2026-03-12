package model

import (
	"net/url"
	"slices"
	"time"

	"github.com/purrice/prawler/internal/enum/seeds"
)

var (
	ValidSeedEvent = []string{seeds.SeedProduced}
)

type SeedEvent struct {
	EventType *string    `json:"event_type,omitempty"`
	Seed      *string    `json:"seed,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

func (s SeedEvent) IsValid() error {
	if s.EventType == nil || s.Seed == nil || s.Timestamp == nil {
		return ErrMissingField
	}

	if *s.Seed == "" {
		return ErrInvalidSeed
	}

	if !slices.Contains(ValidSeedEvent, *s.EventType) {
		return ErrInvalidEventType
	}

	return nil
}

func (s SeedEvent) GetSeed() (*url.URL, error) {
	url, err := url.Parse(*s.Seed)

	if err != nil {
		return nil, ErrInvalidSeed
	}

	return url, nil
}
