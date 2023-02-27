# sticker

`ScheduledTicker` provides a ticker similar to [`time.Ticker`](https://pkg.go.dev/time#Ticker) but can be scheduled to start at a specific point in time.

# Usage

```go
ticker := sticker.New(schedRule.rule.FirstStart, rule.RunInterval)
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
