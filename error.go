package framework

import (
	"errors"
	"fmt"
	"sync"
)

type AggregatedError struct {
	lock   sync.RWMutex
	errors []error
}

func (a *AggregatedError) Errorf(format string, m ...any) {
	a.Append(fmt.Errorf(format, m...))
}

func (a *AggregatedError) Append(errs ...error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.errors = append(a.errors, errs...)
}

func (a *AggregatedError) Join() error {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return errors.Join(a.errors...)
}

func (a *AggregatedError) Empty() bool {
	return a.Join() == nil
}
