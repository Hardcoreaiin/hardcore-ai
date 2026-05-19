package debug

import "sync"

var seenLogs = make(map[string]bool)

var dedupeMutex sync.Mutex

func IsDuplicate(msg string) bool {

	dedupeMutex.Lock()
	defer dedupeMutex.Unlock()

	if seenLogs[msg] {
		return true
	}

	seenLogs[msg] = true

	if len(seenLogs) > 1000 {

		seenLogs = make(map[string]bool)
	}

	return false
}
