package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/roboslone/go-framework"
	"gopkg.in/yaml.v3"
)

type CommandConfig struct {
	Commands map[string]framework.CommandModule[any] `yaml:"commands"`
}

func (cfg *CommandConfig) PrintUsage() {
	result := strings.Builder{}

	result.WriteString("\nAvailable modules:\n")

	for name, module := range cfg.Commands {
		result.WriteString(fmt.Sprintf("\t%s\n", name))

		if len(module.Command) > 0 {
			result.WriteString(color.BlackString("\t\t$ %s\n", strings.Join(module.Command, " ")))
		}

		if module.Dir != "" {
			result.WriteString(color.BlackString("\t\t@%s\n", module.Dir))
		}

		if len(module.DependsOn) > 0 {
			result.WriteString(color.BlackString("\t\tdepends on %s\n", strings.Join(module.DependsOn, ", ")))
		}
	}

	fmt.Println(result.String())
}

func main() {
	fs := flag.NewFlagSet("fexec", flag.ContinueOnError)

	configPath := fs.String("c", ".fexec.yaml", "Path to config file")

	flagErr := fs.Parse(os.Args[1:])
	printUsage := errors.Is(flagErr, flag.ErrHelp) || len(os.Args) == 1
	if !printUsage && flagErr != nil {
		log.Fatalf("parsing options: %s", flagErr)
	}

	cfg, err := ParseConfig(*configPath)
	if err != nil {
		log.Fatalf("reading config: %s", err)
	}
	if printUsage {
		cfg.PrintUsage()

		if len(os.Args) == 1 {
			os.Exit(1)
		}

		return
	}

	modules := framework.Modules{}

	for name, module := range cfg.Commands {
		if len(module.Command) == 0 && module.Dir == "" && len(module.Env) == 0 {
			modules[name] = &framework.NoopModule{DependsOn: module.DependsOn}
			continue
		}

		modules[name] = &framework.CommandModule[any]{
			Command:   module.Command,
			Dir:       module.Dir,
			Env:       module.Env,
			DependsOn: module.DependsOn,
		}
	}

	framework.NewApplication[any]("fexec", modules).Main()
}

func ParseConfig(path string) (*CommandConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	cfg := &CommandConfig{}
	if err = yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	return cfg, nil
}
