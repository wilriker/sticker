# sticker

`ScheduledTicker` provides a ticker similar to [`time.Ticker`](https://pkg.go.dev/time#Ticker) but can be scheduled to start at a specific point in time.

# Usage

```go
ticker := sticker.New(schedule.FirstStart, schedule.Interval)
defer ticker.Stop()

for {
    select {
    case update := <-updateTicker:
        ticker.Reset(update.FirstStart, update.Interval)

    case <-ticker.C:
        // Do your work
    }
}
```

Note that the `FirstStart` can be at any point in time. If it happens to be in the past the next correct occurrence of a tick will be calculated.
