package main

import (
	"context"
	"log"
	"os"

	"github.com/roboslone/go-framework"
)

type State struct{}

var App = framework.NewApplication(
	"framework-tools",
	framework.Modules[State]{
		"install": &framework.CommandModule[State]{
			Command: []string{"go", "get"},
		},
		"lint": &framework.CommandModule[State]{
			Command:   []string{"golangci-lint", "run", "--no-config", "."},
			DependsOn: []string{"install"},
		},
		"test": &framework.CommandModule[State]{
			Command:   []string{"go", "test", "./..."},
			DependsOn: []string{"install"},
		},
		"ci": &framework.NoopModule[State]{
			DependsOn: []string{"lint", "test"},
		},
	},
)

func main() {
	err := App.Run(context.Background(), &State{}, os.Args[1:]...)
	if err != nil {
		log.Fatal(err)
	}
}
