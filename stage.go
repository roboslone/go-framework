package framework

import (
	"sync"
)

type Stage struct {
	Conditions map[string]*sync.Cond
}
