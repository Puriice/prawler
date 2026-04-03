package master

import (
	"context"
	"encoding/json"

	"github.com/purrice/prawler/internal/master/status"
	"github.com/purrice/prawler/internal/repository"
)

type MasterNode struct {
	repo repository.MasterRepository
	ctx  context.Context
}

func NewMasterNode(repo repository.MasterRepository, ctx context.Context) MasterNode {
	return MasterNode{repo: repo, ctx: ctx}
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
	}

	return nil
}
