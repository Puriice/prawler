package frontier

import (
	"fmt"

	"github.com/puriice/golibs/pkg/messaging"
	"github.com/purrice/prawler/internal/config"
)

type Client struct {
	broker *messaging.RabbitBroker
	key    string
}

func NewClient(rabbit *messaging.RabbitMQ) (Client, error) {
	cfg := config.GetConfig()
	broker, err := rabbit.NewBroker(cfg.ExchangeName.Frontier)

	if err != nil {
		return Client{}, err
	}

	return Client{
		broker: broker,
		key:    fmt.Sprintf("%s.confirm", cfg.ExchangeName.Frontier),
	}, nil
}

func (c Client) ConfirmCrawled(e ConfirmPayload) error {
	return c.broker.Publish(c.key, e)
}
