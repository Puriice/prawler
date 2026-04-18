package frontier

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/enum"
	"github.com/purrice/prawler/internal/events"
)

type keys struct {
	Register string
	Confirm  string
}

type Client struct {
	broker *messaging.RabbitBroker
	keys   keys
}

func NewClient(rabbit *messaging.RabbitMQ) (Client, error) {
	cfg := config.GetConfig()
	broker, err := rabbit.NewBroker(cfg.ExchangeName.Frontier)

	if err != nil {
		return Client{}, err
	}

	return Client{
		broker: broker,
		keys: keys{
			Register: fmt.Sprintf("%s.uri", cfg.ExchangeName.Frontier),
			Confirm:  fmt.Sprintf("%s.confirm", cfg.ExchangeName.Frontier),
		},
	}, nil
}

func (c Client) Register(uri string, depth int) error {
	now := time.Now()

	payload := events.URIPayload{
		URI:       &uri,
		Depth:     depth,
		Timestamp: &now,
	}

	bytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	return c.broker.Publish(c.keys.Register, events.FrontierEvent{
		Type:    events.FrontierURIRegister,
		Payload: bytes,
	})
}

func (c Client) FailedCrawled() error {
	payload := events.ConfirmPayload{
		Status: &enum.Page.Failed,

		URI:       nil,
		FinalURI:  nil,
		Canonical: nil,

		Depth: 0,

		Timestamp: time.Now(),
	}

	bytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	return c.broker.Publish(c.keys.Register, events.FrontierEvent{
		Type:    events.FrontierCrawlConfirm,
		Payload: bytes,
	})
}

func (c Client) ConfirmCrawled(
	pageUUID string,
	status enum.PageStatus,
	original string,
	final string,
	canonical string,
	depth int,
) error {
	payload := events.ConfirmPayload{
		Status: &status,

		PageUUID: &pageUUID,

		URI:       &original,
		FinalURI:  &final,
		Canonical: &canonical,

		Depth: depth,

		Timestamp: time.Now(),
	}

	bytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	return c.broker.Publish(c.keys.Register, events.FrontierEvent{
		Type:    events.FrontierCrawlConfirm,
		Payload: bytes,
	})
}
