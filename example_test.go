package framework_test

import (
	"context"
	"fmt"
	"time"

	framework "github.com/roboslone/go-framework"
)

// ExampleState describes example application, both configuration and runtime.
type ExampleState struct {
	// configuration
	Interval      time.Duration
	MaxIterations int

	// runtime
	Channel chan int
}

// Sender is a simple module, that sends increasing integers to `Channel` each `Interval`.
type Sender struct{}

func (*Sender) Start(ctx context.Context, s *ExampleState) error {
	go func() {
		defer close(s.Channel)

		ticker := time.NewTicker(s.Interval)
		var i int
		for {
			select {
			case <-ticker.C:
				s.Channel <- i
				i++

				if i >= s.MaxIterations {
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// Receiver is a simple module, that prints values from `Channel`.
// It depends on Sender.
type Receiver struct{}

func (*Receiver) Start(ctx context.Context, s *ExampleState) error {
	for i := range s.Channel {
		fmt.Println(i)
	}
	return nil
}

func (*Receiver) Dependencies(_ context.Context) []string {
	return []string{
		"sender",
	}
}

func ExampleApplication() {
	// app contains all available modules and their dependencies.
	app := framework.NewApplication[ExampleState](
		"counter",
		framework.Modules{
			"sender":   &Sender{},
			"receiver": &Receiver{},
		},
	)

	state := &ExampleState{
		Interval:      75 * time.Millisecond,
		MaxIterations: 3,
		Channel:       make(chan int),
	}

	// Ensure each module satisfies at least one module interface.
	fmt.Println("check error:", app.Check())

	fmt.Println("run error:", app.Run(context.Background(), context.Background(), state, "receiver"))
	// Output: check error: <nil>
	// 0
	// 1
	// 2
	// run error: <nil>
}
