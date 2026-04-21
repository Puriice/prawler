package frontier

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/enum"
	"github.com/purrice/prawler/internal/events"
	"github.com/purrice/prawler/internal/repository"
)

type keys struct {
	Register string
	Confirm  string
	Backoff  string
}

type Client struct {
	context context.Context

	broker *messaging.RabbitBroker
	keys   keys

	website repository.WebsiteRepository
}

type validatable interface {
	IsValid() error
}

func NewClient(context context.Context, rabbit *messaging.RabbitMQ, website repository.WebsiteRepository) (Client, error) {
	cfg := config.GetConfig()
	broker, err := rabbit.NewBroker(cfg.ExchangeName.Frontier)

	if err != nil {
		return Client{}, err
	}

	return Client{
		context: context,

		broker: broker,
		keys: keys{
			Register: fmt.Sprintf("%s.uri", cfg.ExchangeName.Frontier),
			Confirm:  fmt.Sprintf("%s.confirm", cfg.ExchangeName.Frontier),
			Backoff:  fmt.Sprintf("%s.backoff", cfg.ExchangeName.Frontier),
		},

		website: website,
	}, nil
}

func (c Client) sendPayload(key string, eventType events.FrontierEventType, payload validatable) error {
	if err := payload.IsValid(); err != nil {
		return err
	}

	bytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	return c.broker.Publish(key, events.FrontierEvent{
		Type:    eventType,
		Payload: bytes,
	})
}

func (c Client) Register(uri string, depth int, source *events.Source) error {
	now := time.Now()

	payload := events.URIPayload{
		URI:       &uri,
		Revisit:   false,
		Depth:     depth,
		Source:    source,
		Timestamp: &now,
	}

	return c.sendPayload(
		c.keys.Register,
		events.FrontierURIRegister,
		payload,
	)
}

func (c Client) SkipCrawl(
	pageUUID string,
	httpStatus int,
	original string,
	final string,
	depth int,
) error {
	payload := events.ConfirmPayload{
		Status:     enum.Page.Skipped,
		HTTPStatus: httpStatus,

		PageUUID: &pageUUID,

		URI:       &original,
		FinalURI:  final,
		Canonical: "",

		Depth: depth,

		Timestamp: time.Now(),
	}

	if c.website != nil {
		c.website.SetPageStatus(c.context, pageUUID, enum.Page.Skipped)
	}

	return c.sendPayload(
		c.keys.Confirm,
		events.FrontierCrawlConfirm,
		payload,
	)
}

func (c Client) FailedCrawled(
	pageUUID string,
	httpStatus int,
	original string,
	final string,
	depth int,
) error {
	payload := events.ConfirmPayload{
		Status:     enum.Page.Failed,
		HTTPStatus: httpStatus,

		PageUUID: &pageUUID,

		URI:       &original,
		FinalURI:  final,
		Canonical: "",

		Depth: depth,

		Timestamp: time.Now(),
	}

	return c.sendPayload(
		c.keys.Confirm,
		events.FrontierCrawlConfirm,
		payload,
	)
}

func (c Client) ConfirmCrawled(
	pageUUID string,
	httpStatus int,
	original string,
	final string,
	canonical string,
	depth int,
) error {
	payload := events.ConfirmPayload{
		Status:     enum.Page.Parsed,
		HTTPStatus: httpStatus,

		PageUUID: &pageUUID,

		URI:       &original,
		FinalURI:  final,
		Canonical: canonical,

		Depth: depth,

		Timestamp: time.Now(),
	}

	if c.website != nil {
		c.website.SetPageStatus(c.context, pageUUID, enum.Page.Parsed)
	}

	return c.sendPayload(
		c.keys.Confirm,
		events.FrontierCrawlConfirm,
		payload,
	)
}

func (c Client) Backoff(
	pageUUID string,
	url string,
	httpStatus int,
	retryAfter time.Duration,
) error {
	payload := events.BackoffPayload{
		PageUUID: &pageUUID,

		URI:        &url,
		HTTPStatus: httpStatus,
		RetryAfter: retryAfter,

		Timestamp: time.Now(),
	}

	return c.sendPayload(
		c.keys.Backoff,
		events.FrontierBackoff,
		payload,
	)
}
