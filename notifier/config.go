package notifier

import "time"

// Config is sending settings including log settings
type Config struct {
	Enabled          bool
	SendingTimeout   time.Duration
	ResendingTimeout time.Duration
	Senders          []interface{}
	LogFile          string
	LogLevel         string
	FrontURL         string
}
