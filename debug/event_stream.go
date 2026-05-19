package debug

import (
	"sync"
)

var EventChannel chan DebugEvent

type EventProcessor func(DebugEvent)

var processors []EventProcessor

var processorMutex sync.RWMutex

func InitEventStream() {

	EventChannel = make(
		chan DebugEvent,
		1000,
	)
}

func RegisterProcessor(
	processor EventProcessor,
) {

	processorMutex.Lock()
	defer processorMutex.Unlock()

	processors = append(
		processors,
		processor,
	)
}

func StartEventListener() {

	go func() {

		for event := range EventChannel {

			processorMutex.RLock()

			for _, processor := range processors {

				processor(event)
			}

			processorMutex.RUnlock()
		}
	}()
}
