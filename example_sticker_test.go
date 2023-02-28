package sticker_test

import (
	"time"

	"github.com/wilriker/sticker"
)

type update struct {
	FirstStart time.Time
	Interval   time.Duration
}

// This example demonstrates the basic usage pattern for this ticker including updating its ticking behavior.
func Example() {
	firstStart := time.Now().Add(time.Hour)
	interval := time.Minute
	ticker := sticker.New(firstStart, interval)
	updateTicker := make(chan update)
	defer ticker.Stop()
	for {
		select {
		case update := <-updateTicker:
			ticker.Reset(update.FirstStart, update.Interval)

		case <-ticker.C:
			// Do your work
			// Eventually break
		}
	}
}
