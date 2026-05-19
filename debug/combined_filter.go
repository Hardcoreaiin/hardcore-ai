package debug

import (
	"encoding/json"
	"os"
)

func ReplayFiltered(
	filename string,
	level LogLevel,
	eventType string,
) error {

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

		if event.Level == level &&
			event.Type == eventType {

			Render(event)
		}

	}

	return nil
}
