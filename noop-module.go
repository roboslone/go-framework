package framework

import "context"

type NoopModule[State any] struct {
	Module[State]

	DependsOn []string
}

func (m *NoopModule[State]) Dependencies(context.Context) []string {
	return m.DependsOn
}
