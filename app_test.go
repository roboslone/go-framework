package framework_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	framework "github.com/roboslone/go-framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanity(t *testing.T) {
	mod := &TestModule{}

	require.True(t, isDependent(mod))
	require.True(t, isPreparable(mod))
	require.True(t, isStartable(mod))
	require.True(t, isAwaitable(mod))
	require.True(t, isCleanable(mod))
}

func TestBuildTopology(t *testing.T) {
	t.Run("cycle", func(t *testing.T) {
		// a - b - a
		app := framework.NewApplication[TestState](
			t.Name(),
			framework.Modules{
				"a": NewTestModule("b"),
				"b": NewTestModule("a"),
			},
		)

		_, err := app.BuildTopology(t.Context(), "a")
		require.ErrorContains(t, err, "Cycle error")
	})

	t.Run("self-dependency", func(t *testing.T) {
		// a - a
		app := framework.NewApplication[TestState](
			t.Name(),
			framework.Modules{
				"a": NewTestModule("a"),
			},
		)

		_, err := app.BuildTopology(t.Context(), "a")
		require.ErrorContains(t, err, "Cycle error")
	})

	t.Run("linear", func(t *testing.T) {
		// a - b - c
		app := framework.NewApplication[TestState](
			t.Name(),
			framework.Modules{
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
		app := framework.NewApplication[TestState](
			t.Name(),
			framework.Modules{
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
		app := framework.NewApplication[TestState](
			t.Name(),
			framework.Modules{
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

func TestInStageDependencies(t *testing.T) {
	app := framework.NewApplication[TestState](
		t.Name(),
		framework.Modules{
			"a": &TestModule{
				onPrepare: func(ctx context.Context, ts *TestState) error {
					ts.Value = 42
					return nil
				},
				onStart: func(ctx context.Context, ts *TestState) error {
					ts.Value = 69
					return nil
				},
				onWait: func(ctx context.Context, ts *TestState) error {
					ts.Value = 420
					return nil
				},
				onCleanup: func(ctx context.Context, ts *TestState) error {
					ts.Value = 0
					return nil
				},
			},
			"b": &TestModule{
				onPrepare: func(ctx context.Context, ts *TestState) error {
					assert.Equal(t, 42, ts.Value, "prepare")
					return nil
				},
				onStart: func(ctx context.Context, ts *TestState) error {
					assert.Equal(t, 69, ts.Value, "start")
					return nil
				},
				onWait: func(ctx context.Context, ts *TestState) error {
					assert.Equal(t, 420, ts.Value, "start")
					return nil
				},
				onCleanup: func(ctx context.Context, ts *TestState) error {
					assert.Equal(t, 0, ts.Value, "start")
					return nil
				},
				dependencies: []string{"a"},
			},
		},
	)
	require.NoError(t, app.Run(t.Context(), &TestState{}, "b"))
}

func TestErrors(t *testing.T) {
	mFinite := &TestModule{}
	mInfinite := &TestContextBoundModule{}
	app := framework.NewApplication[TestState](
		t.Name(),
		framework.Modules{
			"finite":   mFinite,
			"infinite": mInfinite,
		},
	)

	t.Run("finite", func(t *testing.T) {
		t.Run("none", func(t *testing.T) {
			require.NoError(t, app.Run(t.Context(), &TestState{}, "finite"))
		})

		t.Run("prepare", func(t *testing.T) {
			mFinite.SetErrors(fmt.Errorf("prepare error"), nil, nil, nil)
			require.ErrorContains(t, app.Run(t.Context(), &TestState{}, "finite"), "prepare error")
		})

		t.Run("start", func(t *testing.T) {
			mFinite.SetErrors(nil, fmt.Errorf("start error"), nil, nil)
			require.ErrorContains(t, app.Run(t.Context(), &TestState{}, "finite"), "start error")
		})

		t.Run("wait", func(t *testing.T) {
			mFinite.SetErrors(nil, nil, fmt.Errorf("wait error"), nil)
			require.ErrorContains(t, app.Run(t.Context(), &TestState{}, "finite"), "wait error")
		})

		t.Run("cleanup", func(t *testing.T) {
			mFinite.SetErrors(nil, nil, nil, fmt.Errorf("cleanup error"))
			require.ErrorContains(t, app.Run(t.Context(), &TestState{}, "finite"), "cleanup error")
		})
	})

	t.Run("infinite", func(t *testing.T) {
		t.Run("none", func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			require.NoError(t, app.Run(ctx, &TestState{}, "infinite"))
		})

		t.Run("prepare", func(t *testing.T) {
			mInfinite.SetErrors(fmt.Errorf("prepare error"), nil, nil, nil)
			require.ErrorContains(t, app.Run(t.Context(), &TestState{}, "infinite"), "prepare error")
		})

		t.Run("start", func(t *testing.T) {
			mInfinite.SetErrors(nil, fmt.Errorf("start error"), nil, nil)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			require.ErrorContains(t, app.Run(ctx, &TestState{}, "infinite"), "start error")
		})

		t.Run("wait", func(t *testing.T) {
			mInfinite.SetErrors(nil, nil, fmt.Errorf("wait error"), nil)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			require.ErrorContains(t, app.Run(ctx, &TestState{}, "infinite"), "wait error")
		})

		t.Run("cleanup", func(t *testing.T) {
			mInfinite.SetErrors(nil, nil, nil, fmt.Errorf("cleanup error"))
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			require.ErrorContains(t, app.Run(ctx, &TestState{}, "infinite"), "cleanup error")
		})
	})
}

func TestContext(t *testing.T) {
	app := framework.NewApplication[TestState](t.Name(), framework.Modules{
		"m": &TestModule{
			onPrepare: func(ctx context.Context, _ *TestState) error {
				assert.Equal(t, t.Name(), framework.GetApplicationName(ctx), "prepare")
				assert.Equal(t, "m", framework.GetModuleName(ctx), "prepare")
				return nil
			},
			onStart: func(ctx context.Context, _ *TestState) error {
				assert.Equal(t, t.Name(), framework.GetApplicationName(ctx), "start")
				assert.Equal(t, "m", framework.GetModuleName(ctx), "start")
				return nil
			},
			onWait: func(ctx context.Context, _ *TestState) error {
				assert.Equal(t, t.Name(), framework.GetApplicationName(ctx), "wait")
				assert.Equal(t, "m", framework.GetModuleName(ctx), "wait")
				return nil
			},
			onCleanup: func(ctx context.Context, _ *TestState) error {
				assert.Equal(t, t.Name(), framework.GetApplicationName(ctx), "cleanup")
				assert.Equal(t, "m", framework.GetModuleName(ctx), "cleanup")
				return nil
			},
		},
	})
	require.NoError(t, app.Run(t.Context(), &TestState{}, "m"))
}

func TestCommandModule(t *testing.T) {
	mod := &framework.CommandModule[TestState]{
		Command: []string{"echo", "hello"},
	}

	require.True(t, isDependent(mod))
	require.False(t, isPreparable(mod))
	require.True(t, isStartable(mod))
	require.False(t, isAwaitable(mod))
	require.False(t, isCleanable(mod))

	app := framework.NewApplication[TestState](t.Name(), framework.Modules{
		"cmd": mod,
	})
	require.NoError(t, app.Run(t.Context(), &TestState{}, "cmd"))
}

func TestNoopModule(t *testing.T) {
	mod := &framework.NoopModule{}

	require.True(t, isDependent(mod))
	require.False(t, isPreparable(mod))
	require.False(t, isStartable(mod))
	require.False(t, isAwaitable(mod))
	require.False(t, isCleanable(mod))
}

func TestInvalidModule(t *testing.T) {
	app := framework.NewApplication[TestState](t.Name(), framework.Modules{"alfa": nil})
	for _, err := range []error{
		app.Check(),
		app.Run(t.Context(), &TestState{}),
		app.Run(t.Context(), &TestState{}, "alfa"),
	} {
		require.ErrorContains(t, err, "contains invalid modules")
		require.ErrorContains(t, err, "alfa")
	}
}

func TestGlob(t *testing.T) {
	app := framework.NewApplication[TestState](t.Name(), framework.Modules{
		"a1": NewTestModule(),
		"a2": NewTestModule(),
		"b1": NewTestModule(),
		"b2": NewTestModule(),
		"c1": NewTestModule(),
		"c2": NewTestModule(),
	})

	names, err := app.Glob("a*")
	require.NoError(t, err)
	slices.Sort(names)
	require.EqualValues(t, []string{"a1", "a2"}, names)

	names, err = app.Glob("b*", "c*")
	require.NoError(t, err)
	slices.Sort(names)
	require.EqualValues(t, []string{"b1", "b2", "c1", "c2"}, names)
}
