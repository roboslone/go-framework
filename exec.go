package framework

import "context"

type ExecutionContext struct {
	topology *Topology
	stages   map[StageName]*Semaphore
	err      *AggregatedError
}

func NewExecutionContext(ctx context.Context, topology *Topology, ae *AggregatedError) *ExecutionContext {
	return &ExecutionContext{
		topology: topology,
		stages: map[StageName]*Semaphore{
			StagePrepare: NewSemaphore(),
			StageStart:   NewSemaphore(),
			StageWait:    NewSemaphore(),
			StageCleanup: NewSemaphore(),
		},
		err: ae,
	}
}

func (c *ExecutionContext) Wait() error {
	c.AwaitStage(StageCleanup)
	return c.err.Join()
}

func (c *ExecutionContext) AwaitStage(name StageName) {
	c.stages[name].Wait()
}
