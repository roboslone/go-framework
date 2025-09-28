package framework

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"

	mapset "github.com/deckarep/golang-set/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Application[State any] struct {
	logger  Logger
	name    string
	modules Modules
}

func NewApplication[State any](name string, modules Modules, options ...ApplicationOption[State]) *Application[State] {
	return &Application[State]{
		name:    name,
		modules: modules,
	}
}

func (a *Application[State]) Main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := a.Run(ctx, new(State), os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}

func (a *Application[State]) Run(ctx context.Context, s *State, modules ...string) error {
	if err := a.Check(); err != nil {
		return err
	}

	exec, err := a.Start(ctx, s, modules...)
	if err != nil {
		return err
	}
	return exec.Wait()
}

func (a *Application[State]) Start(ctx context.Context, s *State, modules ...string) (*ExecutionContext, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(applicationContext(ctx, a.name))

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
			func(m any) bool {
				_, ok := m.(Preparable[State])
				return ok
			},
			func(name string, m any) error {
				return m.(Preparable[State]).Prepare(moduleContext(ctx, name), s)
			},
		)

		if ae.Empty() {
			a.runStage(
				exec, StageStart,
				func(m any) bool {
					_, ok := m.(Startable[State])
					return ok
				},
				func(name string, m any) error {
					return m.(Startable[State]).Start(moduleContext(ctx, name), s)
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
			func(m any) bool {
				_, ok := m.(Awaitable[State])
				return ok
			},
			func(name string, m any) error {
				err := m.(Awaitable[State]).Wait(moduleContext(ctx, name), s)
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
			func(m any) bool {
				_, ok := m.(Cleanable[State])
				return ok
			},
			func(name string, m any) error {
				ctx := applicationContext(context.Background(), a.name)
				ctx = moduleContext(ctx, name)
				return m.(Cleanable[State]).Cleanup(ctx, s)
			},
		)
	}()

	return exec, nil
}

func (a *Application[State]) Check() error {
	invalid := mapset.NewSetFromMapKeys(a.modules)

	for name, module := range a.modules {
		if _, ok := module.(Dependent); ok {
			invalid.Remove(name)
			continue
		}
		if _, ok := module.(Preparable[State]); ok {
			invalid.Remove(name)
			continue
		}
		if _, ok := module.(Startable[State]); ok {
			invalid.Remove(name)
			continue
		}
		if _, ok := module.(Awaitable[State]); ok {
			invalid.Remove(name)
			continue
		}
		if _, ok := module.(Cleanable[State]); ok {
			invalid.Remove(name)
			continue
		}
	}

	if !invalid.IsEmpty() {
		sorted := invalid.ToSlice()
		slices.Sort(sorted)
		return fmt.Errorf("application contains invalid modules: %q: %s", a.name, sorted)
	}

	return nil
}

func (a *Application[State]) getLogger() Logger {
	if a.logger == nil {
		return zap.L()
	}
	return a.logger
}
