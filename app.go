package framework

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Application[State any] struct {
	logger  Logger
	name    string
	modules Modules[State]
}

func NewApplication[State any](name string, modules Modules[State], options ...ApplicationOption[State]) *Application[State] {
	return &Application[State]{
		name:    name,
		modules: modules,
	}
}

func (a *Application[State]) Run(ctx context.Context, s *State, modules ...string) error {
	exec, err := a.Start(ctx, s, modules...)
	if err != nil {
		return fmt.Errorf("starting application: %w", err)
	}
	return exec.Wait()
}

func (a *Application[State]) Start(ctx context.Context, s *State, modules ...string) (*ExecutionContext, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	zf := []zap.Field{
		zap.String("framework.application", a.name),
	}

	log := a.getLogger()
	log.Log(zapcore.InfoLevel, "starting application", zf...)

	topology, err := a.BuildTopology(ctx, modules...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("building topology: %q: %s: %w", a.name, modules, err)
	}

	exec := &ExecutionContext{
		ctx:            ctx,
		allModulesDone: make(chan struct{}),
		topology:       topology,
		stages: map[StageName]*Semaphore{
			StagePrepare: NewSemaphore(),
			StageStart:   NewSemaphore(),
			StageWait:    NewSemaphore(),
			StageCleanup: NewSemaphore(),
		},
	}

	go func() {
		defer cancel()

		a.runStage(
			exec, StagePrepare,
			func(_ string, m ModuleInterface[State]) error {
				return m.Prepare(ctx, s)
			},
		)

		if exec.err.Empty() {
			a.runStage(
				exec, StageStart,
				func(_ string, m ModuleInterface[State]) error {
					return m.Start(ctx, s)
				},
			)
		}

		if !exec.err.Empty() {
			// some module failed to either prepare or start
			log.Log(zapcore.InfoLevel, "cancelling application context", zf...)
			cancel()
		}

		tearDownCtx := context.Background()

		allModulesDone := make(chan struct{})
		go func() {
			a.runStage(
				exec, StageWait,
				func(name string, m ModuleInterface[State]) error {
					err := m.Wait(tearDownCtx, s)
					log.Log(
						zapcore.InfoLevel,
						"module completed",
						append(zf, zap.String("framework.module", name), zap.Error(err))...,
					)
					return err
				},
			)
			close(allModulesDone)
		}()

		select {
		case <-allModulesDone:
		case <-ctx.Done():
		}

		a.runStage(
			exec, StageCleanup,
			func(_ string, m ModuleInterface[State]) error {
				return m.Cleanup(tearDownCtx, s)
			},
		)
	}()

	return exec, nil
}

func (a *Application[State]) getLogger() Logger {
	if a.logger == nil {
		return zap.L()
	}
	return a.logger
}
