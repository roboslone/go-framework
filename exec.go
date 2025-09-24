package framework

import "context"

type ExecutionContext struct {
	ctx            context.Context
	allModulesDone chan struct{}
	topology       *Topology
	stages         map[StageName]*Semaphore
	err            AggregatedError
}

func (c *ExecutionContext) Wait() error {
	select {
	case <-c.ctx.Done():
	case <-c.allModulesDone:
	}
	return c.err.Join()
}

func (c *ExecutionContext) AwaitStage(name StageName) {
	c.stages[name].Wait()
}
