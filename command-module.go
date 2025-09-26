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
	Module[State]

	Command []string
	Dir     string
	Env     []string
}

func (m *CommandModule[State]) Start(ctx context.Context, s *State) error {
	str := strings.Join(m.Command, " ")

	args := make([]string, 0, len(m.Command)-1)
	for _, s := range m.Command[1:] {
		args = append(args, os.ExpandEnv(s))
	}

	cmd := exec.CommandContext(ctx, m.Command[0], args...)
	cmd.Dir = m.Dir
	cmd.Env = append(os.Environ(), m.Env...)

	start := time.Now()
	out, err := cmd.CombinedOutput()
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

	color.Black("$ %s", str)

	if err != nil {
		fmt.Println()
		color.Red(err.Error())
		fmt.Println(string(out))
	} else if len(out) > 0 {
		fmt.Println()
		color.Black(string(out))
	}

	fmt.Println()

	return err
}
