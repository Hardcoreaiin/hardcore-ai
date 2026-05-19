package debug

import (
	"fmt"
	"sync"
)

var RenderedLogs []string

var RenderMutex sync.Mutex

func Render(event DebugEvent) {

	RenderMutex.Lock()
	defer RenderMutex.Unlock()

	log := fmt.Sprintf(
		"#%d [%s] [%s] [%s] %s",
		event.ID,
		event.Timestamp.Format("15:04:05"),
		event.Level,
		event.Type,
		event.Message,
	)

	RenderedLogs = append(
		RenderedLogs,
		log,
	)

	if len(RenderedLogs) > 100 {

		RenderedLogs =
			RenderedLogs[len(RenderedLogs)-100:]
	}
}
