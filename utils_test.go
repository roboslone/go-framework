package framework_test

import (
	"context"
	"log"
	"os"
	"testing"

	framework "github.com/roboslone/go-framework/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type TestState struct {
	Value int
}

type TestModule struct {
	dependencies []string

	onPrepare func(context.Context, *TestState) error
	onStart   func(context.Context, *TestState) error
	onWait    func(context.Context, *TestState) error
	onCleanup func(context.Context, *TestState) error
}

func (m *TestModule) SetErrors(prepare, start, wait, cleanup error) {
	m.onPrepare = func(ctx context.Context, ts *TestState) error {
		return prepare
	}

	m.onStart = func(ctx context.Context, ts *TestState) error {
		return start
	}

	m.onWait = func(ctx context.Context, ts *TestState) error {
		return wait
	}

	m.onCleanup = func(ctx context.Context, ts *TestState) error {
		return cleanup
	}
}

func (m *TestModule) Dependencies(context.Context) []string {
	return m.dependencies
}

func (m *TestModule) Prepare(ctx context.Context, s *TestState) error {
	if m.onPrepare == nil {
		return nil
	}
	return m.onPrepare(ctx, s)
}

func (m *TestModule) Start(ctx context.Context, s *TestState) error {
	if m.onStart == nil {
		return nil
	}
	return m.onStart(ctx, s)
}

func (m *TestModule) Wait(ctx context.Context, s *TestState) error {
	if m.onWait == nil {
		return nil
	}
	return m.onWait(ctx, s)
}

func (m *TestModule) Cleanup(ctx context.Context, s *TestState) error {
	if m.onCleanup == nil {
		return nil
	}
	return m.onCleanup(ctx, s)
}

func NewTestModule(deps ...string) *TestModule {
	return &TestModule{
		dependencies: deps,
	}
}

type TestContextBoundModule struct {
	TestModule
}

func (m *TestContextBoundModule) Wait(ctx context.Context, s *TestState) error {
	<-ctx.Done()
	return m.TestModule.Wait(ctx, s)
}

func TestMain(m *testing.M) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)

	logger, err := cfg.Build()
	if err != nil {
		log.Fatalf("setting up logging: %s", err)
	}
	zap.ReplaceGlobals(logger)

	os.Exit(m.Run())
}

func isDependent(m any) bool {
	_, ok := m.(framework.Dependent)
	return ok
}

func isPreparable(m any) bool {
	_, ok := m.(framework.Preparable[TestState])
	return ok
}

func isStartable(m any) bool {
	_, ok := m.(framework.Startable[TestState])
	return ok
}

func isAwaitable(m any) bool {
	_, ok := m.(framework.Awaitable[TestState])
	return ok
}

func isCleanable(m any) bool {
	_, ok := m.(framework.Cleanable[TestState])
	return ok
}
