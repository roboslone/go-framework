package framework_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	framework "github.com/roboslone/go-framework"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type TestModule struct {
	framework.Module[any]
	Deps []string
}

var _ framework.ModuleInterface[any] = (*TestModule)(nil)

func (m *TestModule) Dependencies(context.Context) []string {
	return m.Deps
}

func NewTestModule(deps ...string) *TestModule {
	return &TestModule{
		Deps: deps,
	}
}

func TestBuildTopology(t *testing.T) {
	t.Run("cycle", func(t *testing.T) {
		// a - b - a
		app := &framework.Application[any]{
			Name: t.Name(),
			Modules: framework.Modules[any]{
				"a": NewTestModule("b"),
				"b": NewTestModule("a"),
			},
		}

		_, err := app.BuildTopology(t.Context(), "a")
		require.ErrorContains(t, err, "Cycle error")
	})

	t.Run("self-dependency", func(t *testing.T) {
		// a - a
		app := &framework.Application[any]{
			Name: t.Name(),
			Modules: framework.Modules[any]{
				"a": NewTestModule("a"),
			},
		}

		_, err := app.BuildTopology(t.Context(), "a")
		require.ErrorContains(t, err, "Cycle error")
	})

	t.Run("linear", func(t *testing.T) {
		// a - b - c
		app := &framework.Application[any]{
			Name: t.Name(),
			Modules: framework.Modules[any]{
				"a": NewTestModule("b"),
				"b": NewTestModule("c"),
				"c": NewTestModule(),
			},
		}

		topology, err := app.BuildTopology(t.Context(), "a")
		require.NoError(t, err)
		require.EqualValues(t, []string{"c", "b", "a"}, topology.OrderedModuleNames)
	})

	t.Run("rhombus", func(t *testing.T) {
		//     b
		//   /  \
		// a     d
		//  \   /
		//    c
		app := &framework.Application[any]{
			Name: t.Name(),
			Modules: framework.Modules[any]{
				"a": NewTestModule(),
				"b": NewTestModule("a"),
				"c": NewTestModule("a"),
				"d": NewTestModule("b", "c"),
			},
		}

		topology, err := app.BuildTopology(t.Context(), "d")
		require.NoError(t, err)
		require.EqualValues(t, []string{"a", "b", "c", "d"}, topology.OrderedModuleNames)
	})

	t.Run("composite", func(t *testing.T) {
		//    c - d         h
		//  /      \      /  \
		// a        e - f    i - j
		//  \     /      \  /
		//     b          g
		app := &framework.Application[any]{
			Name: t.Name(),
			Modules: framework.Modules[any]{
				"a": NewTestModule(),
				"b": NewTestModule("a"),
				"c": NewTestModule("a"),
				"d": NewTestModule("c"),
				"e": NewTestModule("b", "d"),
				"f": NewTestModule("e"),
				"g": NewTestModule("f"),
				"h": NewTestModule("f"),
				"i": NewTestModule("h", "g"),
				"j": NewTestModule("i"),
			},
		}

		topology, err := app.BuildTopology(t.Context(), "j")
		require.NoError(t, err)
		require.EqualValues(t, []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, topology.OrderedModuleNames)
	})
}

type State struct {
	// configuration
	Interval time.Duration

	// runtime
	Counter int
}

type CounterIncrementer struct {
	framework.Module[State]
}

func (*CounterIncrementer) Start(ctx context.Context, s *State) error {
	go func() {
		timedLoop(ctx, s.Interval, func() { s.Counter++ })
	}()
	return nil
}

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
	t := time.NewTicker(d).C
	for {
		select {
		case <-t:
			fn()
		case <-ctx.Done():
			return
		}
	}
}

func TestCustomApp(t *testing.T) {
	logger, err := zap.NewProduction()
	require.NoError(t, err)
	zap.ReplaceGlobals(logger)

	a := framework.Application[State]{
		Name: t.Name(),
		Modules: framework.Modules[State]{
			"incrementer": &CounterIncrementer{},
			"printer":     &CounterPrinter{},
		},
	}
	s := &State{
		Interval: 250 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		<-time.After(s.Interval * 5)
		cancel()
	}()

	err = a.Run(ctx, s, "printer")
	require.NoError(t, err)
	require.GreaterOrEqual(t, s.Counter, 4)
}

type DependencyTestModuleA struct {
	framework.Module[State]
}

func (*DependencyTestModuleA) Prepare(ctx context.Context, s *State) error {
	s.Counter = 42
	return nil
}

type DependencyTestModuleB struct {
	framework.Module[State]
}

func (*DependencyTestModuleB) Dependencies(_ context.Context) []string {
	return []string{"a"}
}

func (*DependencyTestModuleB) Prepare(ctx context.Context, s *State) error {
	if s.Counter != 42 {
		return fmt.Errorf("expected counter to be 42, got %d", s.Counter)
	}
	return nil
}

func TestDependencies(t *testing.T) {
	logger, err := zap.NewProduction()
	require.NoError(t, err)
	zap.ReplaceGlobals(logger)

	app := framework.Application[State]{
		Name: t.Name(),
		Modules: framework.Modules[State]{
			"a": &DependencyTestModuleA{},
			"b": &DependencyTestModuleB{},
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-time.After(100 * time.Millisecond)
		cancel()
	}()

	require.NoError(t, app.Run(ctx, &State{}, "b"))
}
