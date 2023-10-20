package framework

import "time"

const (
	// pollInterval defines the interval time for a poll operation.
	PollInterval = 10 * time.Second
	// pollTimeout defines the time after which the poll operation times out.
	PollTimeout = 30 * time.Second
)
