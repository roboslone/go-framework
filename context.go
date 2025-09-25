package framework

import "context"

type ContextKey string

const (
	contextAppName ContextKey = "framework.application"
	contextModName ContextKey = "framework.module"
)

func applicationContext(ctx context.Context, appName string) context.Context {
	ctx = context.WithValue(ctx, contextAppName, appName)
	return ctx
}

func moduleContext(ctx context.Context, modName string) context.Context {
	ctx = context.WithValue(ctx, contextModName, modName)
	return ctx
}

func GetApplicationName(ctx context.Context) string {
	return ctx.Value(contextAppName).(string)
}

func GetModuleName(ctx context.Context) string {
	return ctx.Value(contextModName).(string)
}
