package framework

import "context"

type Dependent interface {
	// Dependencies reference dependency modules by names.
	Dependencies(context.Context) []string
}

type Preparable[State any] interface {
	// Prepare is called in parallel (respecting dependencies) for each requested module.
	Prepare(context.Context, *State) error
}

type Startable[State any] interface {
	// Start is called in parallel (respecting dependencies) for each requested module
	// after all modules are successfully prepared.
	//
	// It's recommended not to block in Start, so consumers can `AwaitStage(framework.StageStart)`
	// and get control back from `Application` as soon as every module has started.
	Start(context.Context, *State) error
}

type Awaitable[State any] interface {
	// Wait is called in parallel (respecting dependencies) for each requested module
	// after application context is cancelled.
	//
	// Wait is called for each module, even if module failed to prepare or start.
	Wait(context.Context, *State) error
}

type Cleanable[State any] interface {
	// Cleanup is called in parallel (respecting dependencies) for each requested module
	// after application context is cancelled.
	//
	// Cleanup is called with a different, non-cancelled context.
	//
	// Cleanup is called for each module, even if module failed to prepare or start.
	Cleanup(context.Context, *State) error
}

type Modules = map[string]any

// ContextBoundModule shoud cease as soon as given context is done.
// It shouldn't block in Start.
type ContextBoundModule[State any] struct {
	Awaitable[State]
}

func (*ContextBoundModule[State]) Wait(ctx context.Context, _ *State) error {
	<-ctx.Done()
	return nil
}
