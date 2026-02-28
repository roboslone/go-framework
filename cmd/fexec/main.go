package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/roboslone/go-framework/v2"
	"gopkg.in/yaml.v3"
)

const (
	discoverMaxDepth = 7
)

var (
	discoverNames = []string{
		".fexec.yaml",
		".fexec.yml",
	}
)

type CommandConfig struct {
	Commands map[string]framework.CommandModule[any] `yaml:"commands"`
}

func (cfg *CommandConfig) PrintUsage(configPath string) {
	result := strings.Builder{}

	result.WriteString(color.BlackString("\n@%s\n", configPath))
	result.WriteString("Available modules:\n")

	for _, name := range slices.Sorted(maps.Keys(cfg.Commands)) {
		result.WriteString(fmt.Sprintf("\t%s\n", name))

		module := cfg.Commands[name]
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

	wd := "."
	configPath := fs.String("c", "", "Path to config file")
	verbose := fs.Bool("v", false, "Show command descriptions & output for successful commands")
	live := fs.Bool("l", false, "Show live output of all commands")

	flagErr := fs.Parse(os.Args[1:])
	printUsage := errors.Is(flagErr, flag.ErrHelp) || len(os.Args) == 1
	if !printUsage && flagErr != nil {
		log.Fatalf("parsing options: %s", flagErr)
	}

	if *configPath == "" {
		var err error
		*configPath, err = DiscoverConfigPath()
		if err != nil {
			log.Fatalf("discovering config path: %s", err)
		}

		wd = filepath.Dir(*configPath)
	}

	cfg, err := ParseConfig(*configPath)
	if err != nil {
		log.Fatalf("reading config: %s", err)
	}
	if printUsage {
		cfg.PrintUsage(*configPath)

		if len(fs.Args()) == 1 {
			os.Exit(1)
		}

		return
	}

	if err = SetupCommonEnv(); err != nil {
		log.Fatalf("setting up common env: %s", err)
	}

	modules := framework.Modules{}
	for name, module := range cfg.Commands {
		if len(module.Command) == 0 && module.Dir == "" && len(module.Env) == 0 {
			modules[name] = &framework.NoopModule{DependsOn: module.DependsOn}
			continue
		}

		if module.Dir == "" {
			module.Dir = wd
		} else if !filepath.IsAbs(module.Dir) {
			module.Dir = filepath.Join(wd, module.Dir)
		}

		m := &framework.CommandModule[any]{
			Command:       module.Command,
			Dir:           module.Dir,
			Env:           module.Env,
			DependsOn:     module.DependsOn,
			ErrorOnOutput: module.ErrorOnOutput,
			Verbose:       module.Verbose,
			Live:          module.Live,
		}

		if verbose != nil {
			m.Verbose = *verbose
		}
		if live != nil {
			m.Live = *live
		}

		modules[name] = m
	}

	framework.NewApplication[any]("fexec", modules).Main(framework.WithArgs(fs.Args()...))
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

func SetupCommonEnv() error {
	for k, v := range map[string]string{
		"NOW": time.Now().Format(time.RFC3339),
	} {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("%q: %w", k, err)
		}
	}
	return nil
}

func DiscoverConfigPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	for depth := 0; depth < discoverMaxDepth; depth++ {
		for _, name := range discoverNames {
			path := fmt.Sprintf("%s/%s", wd, name)

			if _, err = os.Stat(path); err == nil {
				return path, nil
			}
			if !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("stat %q: %w", path, err)
			}
		}
		wd = filepath.Dir(wd)
	}

	return "", fmt.Errorf("file not found (searched for %s)", strings.Join(discoverNames, ", "))
}
