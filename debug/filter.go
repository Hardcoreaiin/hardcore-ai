package debug

import (
	"encoding/json"
	"os"
)

func ReplayByLevel(
	filename string,
	level LogLevel,
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

		if event.Level == level {
			Render(event)
		}

	}

	return nil
}
