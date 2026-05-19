package debug

import (
	"encoding/json"
	"os"
)

func ReplaySession(filename string) error {

	data, err := os.ReadFile(filename)

	if err != nil {
		return err
	}

	var events []DebugEvent

	err = json.Unmarshal(data, &events)

	if err != nil {
		return err
	}

	for _, event := range events {
		Render(event)
	}

	return nil
}
