package framework

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type StageName string

const (
	StagePrepare StageName = "prepare"
	StageStart   StageName = "start"
	StageWait    StageName = "wait"
	StageCleanup StageName = "cleanup"
)

var (
	verbs = map[StageName][2]string{
		StagePrepare: {"prepare", "preparing"},
		StageStart:   {"start", "starting"},
		StageWait:    {"complete", "awaiting"},
		StageCleanup: {"clean up", "cleaning up"},
	}
)

func (a *Application[State]) runStage(
	e *ExecutionContext,
	stage StageName,
	shouldRun func(any) bool,
	payload func(string, any) error,
) {
	defer e.stages[stage].Release()

	log := a.getLogger()
	zf := []zap.Field{
		zap.String("framework.application", a.name),
		zap.String("framework.stage", string(stage)),
	}

	log.Log(zapcore.DebugLevel, "beginning stage", zf...)

	semaphores := make(map[string]*Semaphore)
	for _, n := range e.topology.OrderedModuleNames {
		semaphores[n] = NewSemaphore()
	}

	wg := sync.WaitGroup{}
	for _, name := range e.topology.OrderedModuleNames {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer semaphores[name].Release()

			mf := append(zf, zap.String("framework.module", name))

			if len(e.topology.FullDependencies[name]) > 0 {
				log.Log(
					zapcore.DebugLevel,
					fmt.Sprintf("%s module: waiting for dependencies: %s", verbs[stage][1], e.topology.FullDependencies[name]),
					mf...,
				)

				// wait for dependencies
				for _, d := range e.topology.FullDependencies[name] {
					semaphores[d].Wait()
				}
			}

			// some dependency failed
			if !e.err.Empty() {
				return
			}

			if !shouldRun(a.modules[name]) {
				return
			}

			log.Log(zapcore.InfoLevel, fmt.Sprintf("%s module", verbs[stage][1]), mf...)
			if err := payload(name, a.modules[name]); err != nil {
				log.Log(zapcore.ErrorLevel, fmt.Sprintf("module failed to %s", verbs[stage][0]), append(mf, zap.Error(err))...)
				e.err.Append("%s module: %q: %w", verbs[stage][1], name, err)
			}
		}()
	}
	wg.Wait()
}
