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
		panic(errors.New("non-positive interval for New ScheduledTicker"))
	}
	// Give the channel a 1-element time buffer.
	// If the client falls behind while reading, we drop ticks
	// on the floor until the client catches up.
	c := make(chan time.Time, 1)
	ticker := &ScheduledTicker{
		ticks: c,
		C:     c,
		stop:  make(chan struct{}),
		reset: make(chan time.Time),
	}
	go ticker.loop()
	ticker.Reset(first, interval)
	return ticker
}

// Reset stops a ticker and resets its period to the specified duration.
// The next tick will arrive at time next and then occur regularly at the new period.
// If time next is in the past it will tick at the matching interval started from that point in the past.
func (st *ScheduledTicker) Reset(next time.Time, interval time.Duration) {
	if interval <= 0 {
		panic(errors.New("non-positive interval for ScheduledTicker.Reset"))
	}
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
	defer func() {
		var ntu chan struct{}
		ntu, nextTickUpdated = nextTickUpdated, nil
		close(ntu)
	}()

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
				select {
				case <-st.stop:
					return
				default:
				}
				sendTime(st.ticks, time.Now())
				ticker = time.NewTicker(st.interval)
				nextTick = ticker.C
				if nextTickUpdated != nil {
					nextTickUpdated <- struct{}{}
				}
			})
		case <-nextTickUpdated:
		// NOTE: this case seems unnecessary but is required to have select reevaluate the reference to channel nextTick
		// that was changed as part of calling Reset.

		case t := <-nextTick:
			sendTime(st.ticks, t)
		}
	}
}

func sendTime(ticks chan<- time.Time, tick time.Time) {
	select {
	case ticks <- tick:
	default:
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
