package framework

import (
	"context"
	"fmt"
	"slices"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stevenle/topsort/v2"
)

type Topology struct {
	RequestedModuleNames      []string
	Graph                     *topsort.Graph[string]
	OrderedModuleNames        []string
	ReverseOrderedModuleNames []string
	DirectDependencies        map[string][]string
	FullDependencies          map[string][]string
}

func (a *Application[State]) BuildTopology(ctx context.Context, requested ...string) (*Topology, error) {
	t := &Topology{
		RequestedModuleNames: requested,
		Graph:                topsort.NewGraph[string](),
		DirectDependencies:   make(map[string][]string),
		FullDependencies:     make(map[string][]string),
	}

	// all modules that are required to run `requested`
	resolved := make([]string, 0, len(requested))
	resolved = append(resolved, requested...)

	var finished bool
	for !finished {
		finished = true

		for _, name := range resolved {
			if _, ok := t.DirectDependencies[name]; ok {
				continue
			}

			module, ok := a.modules[name]
			if !ok {
				return nil, fmt.Errorf("module not registered: %q", name)
			}

			t.DirectDependencies[name] = module.Dependencies(ctx)
			for _, d := range t.DirectDependencies[name] {
				if _, ok := t.DirectDependencies[d]; ok {
					continue
				}

				finished = false
				resolved = append(resolved, d)
			}
		}
	}

	resolved = mapset.NewSet(resolved...).ToSlice()
	slices.Sort(resolved)

	for m, deps := range t.DirectDependencies {
		for _, d := range deps {
			if err := t.Graph.AddEdge(m, d); err != nil {
				return nil, fmt.Errorf("defining graph edge: %q -> %q: %w", m, d, err)
			}
		}
	}

	t.OrderedModuleNames = make([]string, 0, len(resolved))
	accounted := mapset.NewSetWithSize[string](len(resolved))

	for _, root := range resolved {
		deps, err := t.Graph.TopSort(root)
		if err != nil {
			return nil, fmt.Errorf("sorting dependencies of %q: %w", root, err)
		}
		t.FullDependencies[root] = deps[:len(deps)-1]

		for _, d := range deps {
			if !accounted.Contains(d) {
				t.OrderedModuleNames = append(t.OrderedModuleNames, d)
				accounted.Add(d)
			}
		}
	}

	t.ReverseOrderedModuleNames = make([]string, len(t.OrderedModuleNames))
	copy(t.ReverseOrderedModuleNames, t.OrderedModuleNames)
	slices.Reverse(t.ReverseOrderedModuleNames)

	return t, nil
}
