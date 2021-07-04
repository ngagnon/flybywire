package session

import "time"

type watchdog struct {
	interval time.Duration
	timeout  *time.Timer
}

func newWatchdog(interval time.Duration) *watchdog {
	return &watchdog{
		interval: interval,
		timeout:  time.NewTimer(interval),
	}
}

func (dog *watchdog) reset() {
	if !dog.timeout.Stop() {
		<-dog.timeout.C
	}

	dog.timeout.Reset(dog.interval)
}
