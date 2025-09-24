package framework

type ApplicationOption[State any] func(*Application[State])

func WithLogger[State any](logger Logger) ApplicationOption[State] {
	return func(a *Application[State]) {
		a.logger = logger
	}
}
