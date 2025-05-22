package logging

import (
	"time"
)

// LogMessage is the event payload for a log message
type LogMessage struct {
	ID          string
	Time        time.Time
	Level       string
	Persist     bool          // used when we want to show the message in the status bar
	PersistTime time.Duration // used when we want to show the message in the status bar
	Message     string        `json:"msg"`
	Attributes  []Attr
}

type Attr struct {
	Key   string
	Value string
}
