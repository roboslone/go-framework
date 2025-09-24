package framework

import (
	"context"

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
		return err
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

	ae := NewAggregatedError(a.name)
	topology, err := a.BuildTopology(ctx, modules...)
	if err != nil {
		cancel()
		return nil, ae.Append("building topology: %s: %w", modules, err).Join()
	}
	exec := NewExecutionContext(ctx, topology, ae)

	go func() {
		defer cancel()

		a.runStage(
			exec, StagePrepare,
			func(_ string, m ModuleInterface[State]) error {
				return m.Prepare(ctx, s)
			},
		)

		if ae.Empty() {
			a.runStage(
				exec, StageStart,
				func(_ string, m ModuleInterface[State]) error {
					return m.Start(ctx, s)
				},
			)
		}

		if !ae.Empty() {
			// some module failed to either prepare or start
			log.Log(zapcore.InfoLevel, "cancelling application context", append(zf, zap.Error(ae.Join()))...)
			cancel()
		}

		a.runStage(
			exec, StageWait,
			func(name string, m ModuleInterface[State]) error {
				err := m.Wait(ctx, s)
				log.Log(
					zapcore.InfoLevel,
					"module completed",
					append(zf, zap.String("framework.module", name), zap.Error(err))...,
				)
				return err
			},
		)

		a.runStage(
			exec, StageCleanup,
			func(_ string, m ModuleInterface[State]) error {
				return m.Cleanup(context.Background(), s)
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
