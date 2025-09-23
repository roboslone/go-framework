package framework

import "context"

type ModuleInterface[State any] interface {
	Dependencies(context.Context) []string
	Prepare(context.Context, *State) error
	Start(context.Context, *State) error
	Wait(context.Context) error
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

func (*Module[State]) Wait(context.Context) error {
	return nil
}

func (*Module[State]) Cleanup(context.Context, *State) error {
	return nil
}

func (*Module[State]) Dependencies(context.Context) []string {
	return nil
}
