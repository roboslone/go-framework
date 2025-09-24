package framework

import "context"

type ModuleInterface[State any] interface {
	// Dependencies reference dependency modules by names.
	Dependencies(context.Context) []string

	// Prepare is called in parallel (respecting dependencies) for each requested module.
	Prepare(context.Context, *State) error

	// Start is called in parallel (respecting dependencies) for each requested module
	// after all modules are successfully prepared.
	//
	// It's recommended not to block in Start, so consumers can `AwaitStage(framework.StageStart)`
	// and get control back from `Application` as soon as every module has started.
	Start(context.Context, *State) error

	// Wait is called in parallel (respecting dependencies) for each requested module
	// after application context is cancelled.
	//
	// Wait is called for each module, even if module failed to prepare or start.
	Wait(context.Context, *State) error

	// Cleanup is called in parallel (respecting dependencies) for each requested module
	// after application context is cancelled.
	//
	// Cleanup is called with a different, non-cancelled context.
	//
	// Cleanup is called for each module, even if module failed to prepare or start.
	Cleanup(context.Context, *State) error
}

type Module[State any] struct {
	ModuleInterface[State]
}

type Modules[State any] = map[string]ModuleInterface[State]

var _ ModuleInterface[any] = (*Module[any])(nil)

func (*Module[State]) Prepare(context.Context, *State) error {
	return nil
}

func (*Module[State]) Start(context.Context, *State) error {
	return nil
}

func (*Module[State]) Wait(context.Context, *State) error {
	return nil
}

func (*Module[State]) Cleanup(context.Context, *State) error {
	return nil
}

func (*Module[State]) Dependencies(context.Context) []string {
	return nil
}

// ContextBoundModule shoud cease as soon as given context is done.
// It shouldn't block in Start.
type ContextBoundModule[State any] struct {
	Module[State]
}

func (*ContextBoundModule[State]) Wait(ctx context.Context, _ *State) error {
	<-ctx.Done()
	return nil
}
