package debug

import (
	"encoding/json"
	"os"
	"time"
)

func SaveLogs() error {

	os.MkdirAll("logs", os.ModePerm)

	data, err := json.MarshalIndent(DebugStream, "", "  ")

	if err != nil {
		return err
	}

	filename := "logs/session_" +
		time.Now().Format("20060102_150405") +
		".json"

	err = os.WriteFile(filename, data, 0644)

	if err != nil {
		return err
	}

	return nil
}
