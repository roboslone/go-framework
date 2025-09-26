# go-framework

A way to structure your application modules.

<img width="573" height="435" alt="Screenshot 2025-09-26 at 10 14 57" src="https://github.com/user-attachments/assets/3e67e0f7-dd48-4ca0-98e5-80d216e86749" />

^ Output of internal [tool](https://github.com/roboslone/go-framework/blob/main/.tools/main.go#L13) built on `go-framework`.  
Linting and testing are done in parallel.

## Example

```go
// State describes your application, both configuration and runtime.
// In this example state configures `Interval` for "incrementer" and "printer" modules,
// and also stores `Counter`, which is modified by "incrementer" and read by "printer".
type State {
    // configuration
    Interval time.Duration

    // runtime
    Counter int
}

// CounterIncrementer is a simple module, that increments given `Counter` each `Interval`.
type CounterIncrementer struct {
	framework.Module[State]
}

func (*CounterIncrementer) Start(ctx context.Context, s *State) error {
	go func() {
		timedLoop(ctx, s.Interval, func() { s.Counter++ })
	}()
	return nil
}

// CounterPrinter is a simple module, that prints given `Counter` each `Interval`.
// It depends on CounterIncrementer.
type CounterPrinter struct {
	framework.Module[State]
}

func (*CounterPrinter) Start(ctx context.Context, s *State) error {
	go func() {
		timedLoop(ctx, s.Interval, func() { log.Println(s.Counter) })
	}()
	return nil
}

func (*CounterPrinter) Dependencies(_ context.Context) []string {
	return []string{
		"incrementer",
	}
}

func timedLoop(ctx context.Context, d time.Duration, fn func()) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fn()
		case <-ctx.Done():
			return
		}
	}
}

// App contains all available modules and their dependencies.
var App = framework.NewApplication(
    "counter",
    framework.Modules[State]{
        "incrementer": &CounterIncrementer{},
        "printer":     &CounterPrinter{},
    },
)

// Prepares and starts both `incrementer` and `printer`.
App.Run(context.Background(), "printer")
```
