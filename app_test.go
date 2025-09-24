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

func setupLogging(t *testing.T) {
	logger, err := zap.NewProduction()
	require.NoError(t, err)
	zap.ReplaceGlobals(logger)
}

type TestState struct {
	// configuration
	Interval time.Duration

	// runtime
	Counter int
}

type TestModule struct {
	framework.Module[TestState]

	dependencies []string
	prepareErr   error
	startErr     error
	waitErr      error
	cleanupErr   error
}

func (m *TestModule) Dependencies(context.Context) []string {
	return m.dependencies
}

func (m *TestModule) Prepare(ctx context.Context, s *TestState) error {
	return m.prepareErr
}

func (m *TestModule) Start(ctx context.Context, s *TestState) error {
	return m.startErr
}

func (m *TestModule) Wait(ctx context.Context, s *TestState) error {
	return m.waitErr
}

func (m *TestModule) Cleanup(ctx context.Context, s *TestState) error {
	return m.cleanupErr
}

func NewTestModule(deps ...string) *TestModule {
	return &TestModule{
		dependencies: deps,
	}
}

func TestBuildTopology(t *testing.T) {
	t.Run("cycle", func(t *testing.T) {
		// a - b - a
		app := framework.NewApplication(
			t.Name(),
			framework.Modules[TestState]{
				"a": NewTestModule("b"),
				"b": NewTestModule("a"),
			},
		)

		_, err := app.BuildTopology(t.Context(), "a")
		require.ErrorContains(t, err, "Cycle error")
	})

	t.Run("self-dependency", func(t *testing.T) {
		// a - a
		app := framework.NewApplication(
			t.Name(),
			framework.Modules[TestState]{
				"a": NewTestModule("a"),
			},
		)

		_, err := app.BuildTopology(t.Context(), "a")
		require.ErrorContains(t, err, "Cycle error")
	})

	t.Run("linear", func(t *testing.T) {
		// a - b - c
		app := framework.NewApplication(
			t.Name(),
			framework.Modules[TestState]{
				"a": NewTestModule("b"),
				"b": NewTestModule("c"),
				"c": NewTestModule(),
			},
		)

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
		app := framework.NewApplication(
			t.Name(),
			framework.Modules[TestState]{
				"a": NewTestModule(),
				"b": NewTestModule("a"),
				"c": NewTestModule("a"),
				"d": NewTestModule("b", "c"),
			},
		)

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
		app := framework.NewApplication(
			t.Name(),
			framework.Modules[TestState]{
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
		)

		topology, err := app.BuildTopology(t.Context(), "j")
		require.NoError(t, err)
		require.EqualValues(t, []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, topology.OrderedModuleNames)
	})
}

type CounterIncrementer struct {
	framework.Module[TestState]
}

func (*CounterIncrementer) Start(ctx context.Context, s *TestState) error {
	timedLoop(ctx, s.Interval, func() { s.Counter++ })
	return nil
}

type CounterPrinter struct {
	framework.Module[TestState]
}

func (*CounterPrinter) Start(ctx context.Context, s *TestState) error {
	timedLoop(ctx, s.Interval, func() { log.Println(s.Counter) })
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
	setupLogging(t)

	a := framework.NewApplication(
		t.Name(),
		framework.Modules[TestState]{
			"incrementer": &CounterIncrementer{},
			"printer":     &CounterPrinter{},
		},
	)
	s := &TestState{
		Interval: 250 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		<-time.After(s.Interval * 5)
		cancel()
	}()

	require.NoError(t, a.Run(ctx, s, "printer"))
	require.GreaterOrEqual(t, s.Counter, 4)
}

type DependencyTestModuleA struct {
	framework.Module[TestState]
}

func (*DependencyTestModuleA) Prepare(ctx context.Context, s *TestState) error {
	s.Counter = 42
	return nil
}

type DependencyTestModuleB struct {
	framework.Module[TestState]
}

func (*DependencyTestModuleB) Dependencies(_ context.Context) []string {
	return []string{"a"}
}

func (*DependencyTestModuleB) Prepare(ctx context.Context, s *TestState) error {
	if s.Counter != 42 {
		return fmt.Errorf("expected counter to be 42, got %d", s.Counter)
	}
	return nil
}

func TestDependencies(t *testing.T) {
	setupLogging(t)

	app := framework.NewApplication(
		t.Name(),
		framework.Modules[TestState]{
			"a": &DependencyTestModuleA{},
			"b": &DependencyTestModuleB{},
		},
	)

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-time.After(100 * time.Millisecond)
		cancel()
	}()

	require.NoError(t, app.Run(ctx, &TestState{}, "b"))
}

type TestFiniteModule struct {
	TestModule
	done chan struct{}
}

func (m *TestFiniteModule) Prepare(ctx context.Context, s *TestState) error {
	m.done = make(chan struct{})
	return m.TestModule.Prepare(ctx, s)
}

func (m *TestFiniteModule) Start(ctx context.Context, s *TestState) error {
	close(m.done)
	return m.startErr
}

func (m *TestFiniteModule) Wait(ctx context.Context, s *TestState) error {
	<-m.done
	return m.waitErr
}

func TestFinite(t *testing.T) {
	setupLogging(t)

	mod := &TestFiniteModule{}
	app := framework.NewApplication(
		t.Name(),
		framework.Modules[TestState]{
			"finite": mod,
		},
	)

	require.NoError(t, app.Run(t.Context(), &TestState{}, "finite"))

	mod.prepareErr = fmt.Errorf("prepare error")
	require.ErrorContains(t, app.Run(t.Context(), &TestState{}, "finite"), "prepare error")

	mod.prepareErr = nil
	mod.startErr = fmt.Errorf("start error")
	require.ErrorContains(t, app.Run(t.Context(), &TestState{}, "finite"), "start error")

	mod.startErr = nil
	mod.waitErr = fmt.Errorf("wait error")
	require.ErrorContains(t, app.Run(t.Context(), &TestState{}, "finite"), "wait error")
}
