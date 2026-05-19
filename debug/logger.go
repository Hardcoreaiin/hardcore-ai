package debug

import "time"

var DebugStream []DebugEvent
var EventCounter int

func Log(
	level LogLevel,
	eventType string,
	msg string,
) {

	EventCounter++

	event := DebugEvent{
		ID:        EventCounter,
		Timestamp: time.Now(),
		Level:     level,
		Type:      eventType,
		Message:   msg,
	}

	if IsDuplicate(msg) {
		return
	}

	DebugStream = append(DebugStream, event)

	UpdateStats(level)

	select {

	case EventChannel <- event:

		// Event sent successfully

	default:

		// Queue full
		SystemStats.DroppedEvents++

	}

}
