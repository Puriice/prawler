package master

import (
	"context"
	"encoding/json"

	"github.com/purrice/prawler/internal/master/status"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/repository"
	"github.com/purrice/prawler/internal/robots"
)

type MasterConfig struct {
}

type MasterNode struct {
	repo        repository.MasterRepository
	ctx         context.Context
	robotPraser *robots.RobotParser
}

func NewMasterNode(
	ctx context.Context,
	repository repository.MasterRepository,
	robotPraser *robots.RobotParser,
) MasterNode {
	return MasterNode{
		repo:        repository,
		ctx:         ctx,
		robotPraser: robotPraser,
	}
}

func (m MasterNode) handleReportStatus(payload StatusPayload) error {
	switch *payload.Status {
	case status.Activation:
		return m.repo.AddCrawler(m.ctx, *payload.UUID, *payload.Timestamp)
	case status.Heartbeat:
	case status.Deactivation:
	}

	return nil
}

func (m MasterNode) handleURIRegister(payload model.URIPayload) error {
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
	case StatusReport:
		payload, ok := event.Payload.(StatusPayload)

		if !ok {
			return nil
		}

		if err := payload.IsValid(); err != nil {
			return nil
		}

		return m.handleReportStatus(payload)
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
