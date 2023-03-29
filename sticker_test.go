package sticker

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// NOTE: this next method is a straight copy from stdlib/time/tick_test.go, only adjusted New-method name
func TestTicker(t *testing.T) {
	// We want to test that a ticker takes as much time as expected.
	// Since we don't want the test to run for too long, we don't
	// want to use lengthy times. This makes the test inherently flaky.
	// Start with a short time, but try again with a long one if the
	// first test fails.

	baseCount := 10
	baseDelta := 20 * time.Millisecond

	// On Darwin ARM64 the tick frequency seems limited. Issue 35692.
	if (runtime.GOOS == "darwin" || runtime.GOOS == "ios") && runtime.GOARCH == "arm64" {
		// The following test will run ticker count/2 times then reset
		// the ticker to double the duration for the rest of count/2.
		// Since tick frequency is limited on Darwin ARM64, use even
		// number to give the ticks more time to let the test pass.
		// See CL 220638.
		baseCount = 6
		baseDelta = 100 * time.Millisecond
	}

	var errs []string
	logErrs := func() {
		for _, e := range errs {
			t.Log(e)
		}
	}

	for _, test := range []struct {
		count int
		delta time.Duration
	}{{
		count: baseCount,
		delta: baseDelta,
	}, {
		count: 8,
		delta: 1 * time.Second,
	}} {
		count, delta := test.count, test.delta
		ticker := New(time.Now(), delta)
		t0 := time.Now()
		for i := 0; i < count/2; i++ {
			<-ticker.C
		}
		ticker.Reset(time.Now(), delta*2)
		for i := count / 2; i < count; i++ {
			<-ticker.C
		}
		ticker.Stop()
		t1 := time.Now()
		dt := t1.Sub(t0)
		target := 3 * delta * time.Duration(count/2)
		slop := target * 3 / 10
		if dt < target-slop || dt > target+slop {
			errs = append(errs, fmt.Sprintf("%d %s ticks then %d %s ticks took %s, expected [%s,%s]", count/2, delta, count/2, delta*2, dt, target-slop, target+slop))
			if dt > target+slop {
				// System may be overloaded; sleep a bit
				// in the hopes it will recover.
				time.Sleep(time.Second / 2)
			}
			continue
		}
		// Now test that the ticker stopped.
		time.Sleep(2 * delta)
		select {
		case <-ticker.C:
			errs = append(errs, "Ticker did not shut down")
			continue
		default:
			// ok
		}

		// Test passed, so all done.
		if len(errs) > 0 {
			t.Logf("saw %d errors, ignoring to avoid flakiness", len(errs))
			logErrs()
		}

		return
	}

	t.Errorf("saw %d errors", len(errs))
	logErrs()
}

// Test that NewTicker panics when given a duration less than zero.
func TestNewTickerLtZeroDuration(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("expected panic but got none")
		}
	}()
	New(time.Time{}, -1)
}

// Test that Ticker.Reset panics when given a duration less than zero.
func TestTickerResetLtZeroDuration(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("Ticker.Reset(0) should have panicked")
		}
	}()
	tk := New(time.Time{}, time.Second)
	tk.Reset(time.Time{}, 0)
}

func TestStopAfterReset(t *testing.T) {
	ticker := New(time.Now().UTC().Add(time.Hour), 1)
	ticker.Stop()
}

func TestScheduledTicks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	startAt := time.Now().Add(time.Second)
	tk := New(startAt, time.Minute)
	defer tk.Stop()
	start := time.Now()
	<-tk.C
	passed := time.Since(start)
	if passed < time.Second {
		t.Errorf("tick too early, after %v", passed)
	} else if passed > time.Second+3*time.Millisecond {
		t.Errorf("tick too late, after %v", passed)
	}

}

func TestDropOnSlowClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	startAt := time.Now()
	interval := 100 * time.Millisecond
	st := New(startAt, interval)
	expextedTick := startAt.Add(interval)
	defer st.Stop()
	time.Sleep(5 * interval)
	ticked := <-st.C
	if ticked.Sub(expextedTick) >= 3*time.Millisecond {
		t.Errorf("want: %v, got: %v", expextedTick, ticked)
	}
}

func TestNextRun(t *testing.T) {
	cases := []struct {
		name       string
		firstStart time.Time
		interval   time.Duration
		expected   time.Time
	}{
		{
			name:       "distantFuture",
			firstStart: time.Date(2345, 1, 1, 0, 0, 0, 0, time.UTC),
			interval:   time.Minute,
			expected:   time.Date(2345, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:       "startInPast",
			firstStart: time.Now().UTC().Truncate(24 * time.Hour),
			interval:   15 * time.Minute,
			expected: time.Now().UTC().
				Add(15 * (30 * time.Second)). // NOTE: this is basically to force Round below into a Ceil
				Round(15 * time.Minute),
		},
		// NOTE: the following test is hard to calculate a rolling-result correctly
		// {
		// 	"odd",
		// 	time.Date(2021, 11, 30, 14, 48, 0, 0, time.UTC),
		// 	17 * time.Hour,
		// 	time.Date(2021, 11, 30, 14, 48, 0, 0, time.UTC).Add(17 * time.Hour),
		// },
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			firstRun := nextRun(tc.firstStart, tc.interval)
			if !firstRun.Equal(tc.expected) {
				t.Errorf("expected %v, but got %v", tc.expected, firstRun)
			}
		})
	}
}
