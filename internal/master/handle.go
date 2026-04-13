package master

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/origin"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
)

type MasterConfig struct {
}

type MasterNode struct {
	config      *config.Config
	repo        repository.MasterRepository
	ctx         context.Context
	blacklists  *config.Blacklists
	robotPraser *robots.RobotParser
}

func NewMasterNode(
	ctx context.Context,
	db *pgxpool.Pool,
	robotPraser *robots.RobotParser,
) MasterNode {
	repo := repository.NewPostgresMasterRepository(db)
	blacklistRepo := repository.NewPostgresBlacklistRepository(db)
	blacklists := config.NewBlacklist(blacklistRepo)

	config := config.GetConfig()

	return MasterNode{
		config:      config,
		repo:        repo,
		ctx:         ctx,
		blacklists:  blacklists,
		robotPraser: robotPraser,
	}
}

func (m MasterNode) handleURIRegister(payload model.URIPayload) error {
	url, err := url.Parse(*payload.URI)

	if err != nil {
		return nil // Error parsing url return nil because we don't want a retry
	}

	origin := origin.GetOrigin(*url)

	if m.blacklists.Contains(origin.String()) {
		return nil
	}

	rbs, err := m.robotPraser.Parse(*url)

	if errors.Is(err, robots.ErrNotAllowed) {
		m.blacklists.Add(origin.String())
		return nil
	} else if err != nil {
		return nil
	}

	if !rbs.IsAllow(m.config.UserAgent, *payload.URI) {
		return nil
	}

	return nil
}

func (m MasterNode) Handle(data []byte) error {
	var event Event

	err := json.Unmarshal(data, &event)

	if err != nil {
		return err
	}

	if err := event.IsValid(); err != nil {
		return err
	}

	switch event.Type {
	case URIRegister:
		payload, ok := event.Payload.(model.URIPayload)

		if !ok {
			return nil
		}

		if err := payload.IsValid(); err != nil {
			return nil
		}

		return m.handleURIRegister(payload)
	}

	return nil
}
