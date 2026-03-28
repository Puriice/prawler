package master

import "encoding/json"

func Handle(data []byte) error {
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
