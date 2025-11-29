package framework

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
)

type CommandModule[State any] struct {
	Command   []string `yaml:"command"`
	Dir       string   `yaml:"dir"`
	Env       []string `yaml:"env"`
	DependsOn []string `yaml:"dependencies"`
	Verbose   bool     `yaml:"verbose"`
	Live      bool     `yaml:"live"`
}

func (m *CommandModule[State]) Start(ctx context.Context, _ *State) error {
	str := strings.Join(m.Command, " ")

	args := make([]string, 0, len(m.Command)-1)
	for _, s := range m.Command[1:] {
		args = append(args, os.ExpandEnv(s))
	}

	cmd := exec.CommandContext(ctx, m.Command[0], args...)
	cmd.Dir = m.Dir
	cmd.Env = append(os.Environ(), m.Env...)

	if m.Verbose {
		fmt.Printf(
			"%s %s %s\n",
			color.BlueString("↪︎"),
			GetModuleName(ctx),
			color.BlackString("starting..."),
		)
	}

	start := time.Now()

	var out []byte
	var err error
	if m.Live {
		cmd.Stdout = NewPrefixedWriter(os.Stdout, color.BlackString("[%s] ", GetModuleName(ctx)))
		cmd.Stderr = NewPrefixedWriter(os.Stderr, color.BlackString("[%s] ", GetModuleName(ctx)))
		err = cmd.Run()
	} else {
		out, err = cmd.CombinedOutput()
	}

	duration := time.Since(start).Round(time.Millisecond).String()

	if err != nil {
		fmt.Printf(
			"%s %s %s\n",
			color.RedString("❌"),
			GetModuleName(ctx),
			color.BlackString(duration),
		)
	} else {
		fmt.Printf(
			"%s %s %s\n",
			color.GreenString("✓"),
			GetModuleName(ctx),
			color.BlackString(duration),
		)
	}

	if m.Verbose {
		color.Black("$ %s", str)
	}

	if err != nil {
		if !m.Verbose {
			color.Black("$ %s", str)
		}
		color.Red(err.Error())
		fmt.Println(string(out))
	} else if len(out) > 0 {
		if m.Verbose {
			fmt.Println()
			color.Black(string(out))
		}
	}

	if m.Verbose {
		fmt.Println()
	}

	return err
}

func (m *CommandModule[State]) Dependencies(context.Context) []string {
	return m.DependsOn
}
