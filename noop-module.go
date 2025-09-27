package framework

import "context"

type NoopModule struct {
	DependsOn []string
}

func (m *NoopModule) Dependencies(context.Context) []string {
	return m.DependsOn
}
