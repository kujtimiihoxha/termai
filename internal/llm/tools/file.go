package tools

import (
	"sync"
	"time"

	"github.com/kujtimiihoxha/termai/internal/config"
)

// File record to track when files were read/written
type fileRecord struct {
	path      string
	readTime  time.Time
	writeTime time.Time
}

var (
	fileRecords     = make(map[string]fileRecord)
	fileRecordMutex sync.RWMutex
)

func removeWorkingDirectoryPrefix(path string) string {
	wd := config.WorkingDirectory()
	if len(path) > len(wd) && path[:len(wd)] == wd {
		return path[len(wd)+1:]
	}
	return path
}

func recordFileRead(path string) {
	fileRecordMutex.Lock()
	defer fileRecordMutex.Unlock()

	record, exists := fileRecords[path]
	if !exists {
		record = fileRecord{path: path}
	}
	record.readTime = time.Now()
	fileRecords[path] = record
}

func getLastReadTime(path string) time.Time {
	fileRecordMutex.RLock()
	defer fileRecordMutex.RUnlock()

	record, exists := fileRecords[path]
	if !exists {
		return time.Time{}
	}
	return record.readTime
}

func recordFileWrite(path string) {
	fileRecordMutex.Lock()
	defer fileRecordMutex.Unlock()

	record, exists := fileRecords[path]
	if !exists {
		record = fileRecord{path: path}
	}
	record.writeTime = time.Now()
	fileRecords[path] = record
}
