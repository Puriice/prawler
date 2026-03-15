package file

import (
	"encoding/json"
	"os"
)

func LoadJson(path string, payload any) error {
	content, err := os.ReadFile(path)

	if err != nil {
		return err
	}

	return json.Unmarshal(content, &payload)
}
