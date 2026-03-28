package master

import (
	"encoding/json"

	"github.com/purrice/prawler/internal/repository"
)

type MasterNode struct {
	repo repository.MasterRepository
}

func NewMasterNode(repo repository.MasterRepository) MasterNode {
	return MasterNode{repo: repo}
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

	return nil
}
