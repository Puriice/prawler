package frontier

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/model"
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

func (c Client) Register(uri string) error {
	now := time.Now()

	payload := model.URIPayload{
		URI:       &uri,
		Timestamp: &now,
	}

	bytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	return c.broker.Publish(c.keys.Register, Event{
		Type:    EventURIRegister,
		Payload: bytes,
	})
}

func (c Client) ConfirmCrawled(
	original string,
	final string,
	canonical string,
	depth int,
) error {
	payload := ConfirmPayload{
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

	return c.broker.Publish(c.keys.Register, Event{
		Type:    EventCrawlConfirm,
		Payload: bytes,
	})
}
