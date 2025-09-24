package framework

import (
	"errors"
	"fmt"
	"sync"
)

type AggregatedError struct {
	appName string
	lock    sync.RWMutex
	errors  []error
}

func NewAggregatedError(appName string) *AggregatedError {
	return &AggregatedError{
		appName: appName,
	}
}

func (a *AggregatedError) Append(format string, m ...any) *AggregatedError {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.errors = append(a.errors, fmt.Errorf(format, m...))
	return a
}

func (a *AggregatedError) Join() error {
	a.lock.RLock()
	defer a.lock.RUnlock()

	err := errors.Join(a.errors...)
	if err == nil {
		return nil
	}
	return fmt.Errorf("%q: %w", a.appName, err)
}

func (a *AggregatedError) Empty() bool {
	return a.Join() == nil
}
