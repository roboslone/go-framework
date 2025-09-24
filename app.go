package framework

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Application[State any] struct {
	logger     Logger
	name       string
	modules    Modules[State]
	stagesLock sync.RWMutex
	stages     map[StageName]*Semaphore
}

func NewApplication[State any](name string, modules Modules[State], options ...ApplicationOption[State]) *Application[State] {
	return &Application[State]{
		name:    name,
		modules: modules,
	}
}

func (a *Application[State]) Run(ctx context.Context, s *State, modules ...string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	a.initStages()

	zf := []zap.Field{
		zap.String("framework.application", a.name),
	}

	log := a.getLogger()
	log.Log(zapcore.InfoLevel, "starting application", zf...)

	topology, err := a.BuildTopology(ctx, modules...)
	if err != nil {
		return fmt.Errorf("building topology: %q: %s: %w", a.name, modules, err)
	}

	ae := &AggregatedError{}

	a.runStage(
		StagePrepare, topology, ae,
		func(_ string, m ModuleInterface[State]) error {
			return m.Prepare(ctx, s)
		},
	)

	if ae.Empty() {
		a.runStage(
			StageStart, topology, ae,
			func(_ string, m ModuleInterface[State]) error {
				return m.Start(ctx, s)
			},
		)
	}

	if !ae.Empty() {
		// some module failed to either prepare or start
		log.Log(zapcore.InfoLevel, "cancelling application context", zf...)
		cancel()
	}

	tearDownCtx := context.Background()

	allModulesDone := make(chan struct{})
	go func() {
		a.runStage(
			StageWait, topology, ae,
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
		StageCleanup, topology, ae,
		func(_ string, m ModuleInterface[State]) error {
			return m.Cleanup(tearDownCtx, s)
		},
	)

	return ae.Join()
}

func (a *Application[State]) AwaitStage(name StageName) {
	a.stagesLock.RLock()
	s := a.stages[name]
	a.stagesLock.RUnlock()
	s.Wait()
}

func (a *Application[State]) getLogger() Logger {
	if a.logger == nil {
		return zap.L()
	}
	return a.logger
}

func (a *Application[State]) initStages() {
	a.stagesLock.Lock()
	defer a.stagesLock.Unlock()
	a.stages = map[StageName]*Semaphore{
		StagePrepare: NewSemaphore(),
		StageStart:   NewSemaphore(),
		StageWait:    NewSemaphore(),
		StageCleanup: NewSemaphore(),
	}
}
