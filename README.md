# go-framework

A way to structure your application modules.

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
        // timedLoop implementation can be found in `app_test.go`
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
        // timedLoop implementation can be found in `app_test.go`
		timedLoop(ctx, s.Interval, func() { log.Println(s.Counter) })
	}()
	return nil
}

func (*CounterPrinter) Dependencies(_ context.Context) []string {
	return []string{
		"incrementer",
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