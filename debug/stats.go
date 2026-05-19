package debug

import (
	"fmt"
	"time"
)

type Stats struct {
	TotalEvents  int
	InfoCount    int
	WarningCount int
	ErrorCount   int
	DebugCount   int

	EventsPerSecond float64

	QueueUsage    int
	QueueCapacity int
	QueuePressure float64

	QueueWarning bool

	DroppedEvents int
}

var RecentEvents []string

var SystemStats Stats

var StartTime time.Time

func InitStats() {
	StartTime = time.Now()
}

func AddRecentEvent(event DebugEvent) {

	log := fmt.Sprintf(
		"#%d [%s] [%s] %s",
		event.ID,
		event.Level,
		event.Type,
		event.Message,
	)

	RecentEvents = append(
		RecentEvents,
		log,
	)

	// Keep only latest 10 events

	if len(RecentEvents) > 10 {

		RecentEvents =
			RecentEvents[len(RecentEvents)-10:]

	}

}

func UpdateStats(level LogLevel) {

	SystemStats.TotalEvents++

	switch level {

	case INFO:
		SystemStats.InfoCount++

	case WARNING:
		SystemStats.WarningCount++

	case ERROR:
		SystemStats.ErrorCount++

	case DEBUG:
		SystemStats.DebugCount++

	}

	UpdateEPS()

	UpdateQueueStats()
}

func UpdateEPS() {

	elapsed := time.Since(StartTime).Seconds()

	if elapsed > 0 {

		SystemStats.EventsPerSecond =
			float64(SystemStats.TotalEvents) / elapsed

	}

}

func UpdateQueueStats() {

	SystemStats.QueueUsage =
		len(EventChannel)

	SystemStats.QueueCapacity =
		cap(EventChannel)

	if SystemStats.QueueCapacity > 0 {

		SystemStats.QueuePressure =
			(float64(SystemStats.QueueUsage) /
				float64(SystemStats.QueueCapacity)) * 100

	}

	if SystemStats.QueuePressure >= 80 {

		SystemStats.QueueWarning = true

	} else {

		SystemStats.QueueWarning = false

	}

}

func PrintStats() {

	fmt.Println("\n========== SYSTEM STATS ==========")

	fmt.Println("Total Events:",
		SystemStats.TotalEvents)

	fmt.Println("Info Events:",
		SystemStats.InfoCount)

	fmt.Println("Warning Events:",
		SystemStats.WarningCount)

	fmt.Println("Error Events:",
		SystemStats.ErrorCount)

	fmt.Println("Debug Events:",
		SystemStats.DebugCount)

	fmt.Printf(
		"Events Per Second: %.2f\n",
		SystemStats.EventsPerSecond,
	)

	fmt.Println("Queue Usage:",
		SystemStats.QueueUsage)

	fmt.Println("Queue Capacity:",
		SystemStats.QueueCapacity)

	fmt.Printf(
		"Queue Pressure: %.2f%%\n",
		SystemStats.QueuePressure,
	)

	if SystemStats.QueueWarning {

		fmt.Println(
			"WARNING: Queue pressure critical!",
		)

	}

	fmt.Println("Dropped Events:",
		SystemStats.DroppedEvents)

}
