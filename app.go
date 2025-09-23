package framework

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Application[State any] struct {
	Name    string
	Logger  Logger
	Modules Modules[State]
}

func (a *Application[State]) Run(ctx context.Context, s *State, modules ...string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	zf := []zap.Field{
		zap.String("framework.application", a.Name),
	}

	log := a.getLogger()
	log.Log(zapcore.InfoLevel, "starting application", zf...)

	topology, err := a.BuildTopology(ctx, modules...)
	if err != nil {
		return fmt.Errorf("building topology: %q: %s: %w", a.Name, modules, err)
	}

	var ae AggregatedError

	a.runStage(
		[2]string{"prepare", "preparing"},
		topology,
		&ae,
		func(m ModuleInterface[State]) error {
			return m.Prepare(ctx, s)
		},
	)

	if ae.Empty() {
		a.runStage(
			[2]string{"start", "starting"},
			topology,
			&ae,
			func(m ModuleInterface[State]) error {
				return m.Start(ctx, s)
			},
		)
	}

	if !ae.Empty() {
		// some module failed to either prepare or start
		log.Log(zapcore.InfoLevel, "cancelling application context", zf...)
		cancel()
	}

	<-ctx.Done()

	tearDownCtx := context.Background()

	a.runStage(
		[2]string{"wait", "awaiting"},
		topology,
		&ae,
		func(m ModuleInterface[State]) error {
			return m.Wait(tearDownCtx)
		},
	)

	a.runStage(
		[2]string{"clean up", "cleaning up"},
		topology,
		&ae,
		func(m ModuleInterface[State]) error {
			return m.Cleanup(tearDownCtx, s)
		},
	)

	return ae.Join()
}

func (a *Application[State]) getLogger() Logger {
	if a.Logger == nil {
		return zap.L()
	}
	return a.Logger
}

func (a *Application[State]) runStage(
	verbs [2]string,
	t *Topology[State],
	ae *AggregatedError,
	payload func(ModuleInterface[State]) error,
) {
	log := a.getLogger()

	zf := []zap.Field{
		zap.String("framework.application", a.Name),
	}

	semaphores := make(map[string]*Semaphore)
	for _, n := range t.OrderedModuleNames {
		semaphores[n] = NewSemaphore()
	}

	wg := sync.WaitGroup{}
	for _, name := range t.OrderedModuleNames {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer semaphores[name].Release()

			mf := append(zf, zap.String("framework.module", name))

			log.Log(
				zapcore.DebugLevel,
				fmt.Sprintf("%s module: waiting for dependencies: %s", verbs[1], t.FullDependencies[name]),
				mf...,
			)

			// wait for dependencies
			for _, d := range t.FullDependencies[name] {
				semaphores[d].Wait()
			}

			// some dependency failed
			if !ae.Empty() {
				return
			}

			log.Log(zapcore.InfoLevel, fmt.Sprintf("%s module", verbs[1]), mf...)
			if err := payload(a.Modules[name]); err != nil {
				log.Log(zapcore.ErrorLevel, fmt.Sprintf("module failed to %s", verbs[0]), append(mf, zap.Error(err))...)
				ae.Errorf("%s module: %q.%q: %w", verbs[1], a.Name, name, err)
			}
		}()
	}
	wg.Wait()
}
