// Package sticker provides a ticker implementation similar to [time.Ticker] that can be configured to start ticking at a given point in time.
//
// It tries to mimick the API of time.Ticker as much as possible to be usable as a near drop-in replacement.
package sticker

import (
	"errors"
	"time"
)

// ScheduledTicker provides a ticker similar to [time.Ticker] but can be scheduled to start at a specific point in time.
type ScheduledTicker struct {
	C <-chan time.Time // The channel on which the ticks are delivered.

	ticks    chan time.Time
	reset    chan time.Time
	stop     chan struct{}
	interval time.Duration
}

// New returns a new ScheduleTicker that starts
// ticking at time first in the given interval.
// The duration interval must be greater than zero; if not, New will
// panic. Stop the ticker to release associated resources.
func New(first time.Time, interval time.Duration) *ScheduledTicker {
	if interval <= 0 {
		panic(errors.New("non-positive interval for NewScheduledTicker"))
	}
	ticker := &ScheduledTicker{
		ticks: make(chan time.Time, 1),
		stop:  make(chan struct{}),
		reset: make(chan time.Time),
	}
	ticker.C = ticker.ticks
	go ticker.loop()
	ticker.Reset(first, interval)
	return ticker
}

// Reset stops a ticker and resets its period to the specified duration.
// The next tick will arrive at time next and then occur regularly at the new period.
// If time next is in the past it will tick at the matching interval started from that point in the past.
func (st *ScheduledTicker) Reset(next time.Time, interval time.Duration) {
	st.interval = interval
	st.reset <- next
}

// Stop turns off a ticker. After Stop, no more ticks will be sent.
// Stop does not close the channel, to prevent a concurrent goroutine
// reading from the channel from seeing an erroneous "tick".
func (st *ScheduledTicker) Stop() {
	close(st.stop)
	close(st.reset)
}

func (st *ScheduledTicker) loop() {
	var nextTick <-chan time.Time
	var ticker *time.Ticker
	var resetTimer *time.Timer

	nextTickUpdated := make(chan struct{})
	defer close(nextTickUpdated)

	stopTickerTimer := func() {
		nextTick = nil
		if resetTimer != nil {
			resetTimer.Stop()
			resetTimer = nil
		}
		if ticker != nil {
			ticker.Stop()
			ticker = nil
		}
	}
	defer stopTickerTimer()
	for {
		select {
		case <-st.stop:
			return
		case nextStart := <-st.reset:
			stopTickerTimer()
			resetTimer = time.AfterFunc(time.Until(nextRun(nextStart, st.interval)), func() {
				st.ticks <- time.Now().UTC()
				ticker = time.NewTicker(st.interval)
				nextTick = ticker.C
				nextTickUpdated <- struct{}{}
			})
		case <-nextTickUpdated:
		// NOTE: this case seems unnecessary but is required because when nextTick is updated as part of Reset()
		// select won't automatically recognize this and we need to take one loop to get the
		// reference updated for the appropriate case.
		case t := <-nextTick:
			st.ticks <- t
		}
	}
}

// nextRun calculates the next point in time starting from firstStart re-occurring at interval.
func nextRun(firstStart time.Time, interval time.Duration) time.Time {
	// Simple case: we start first time in the future
	if time.Now().UTC().Before(firstStart) {
		return firstStart
	}
	// Now we have to calculate the next run in interval since first start
	pastIterations := time.Since(firstStart) / interval
	return firstStart.Add((pastIterations + 1) * interval)
}
