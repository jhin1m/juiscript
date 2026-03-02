package system

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

// DefaultTimeout for command execution.
const DefaultTimeout = 30 * time.Second

// Executor runs system commands safely.
// Interface allows mocking in tests without needing root.
type Executor interface {
	// Run executes a command and returns combined output.
	Run(ctx context.Context, name string, args ...string) (string, error)
	// RunWithInput executes a command with stdin data.
	RunWithInput(ctx context.Context, input string, name string, args ...string) (string, error)
}

// execImpl is the real executor that calls os/exec.
type execImpl struct {
	logger *slog.Logger
}

// NewExecutor creates a production executor with logging.
func NewExecutor(logger *slog.Logger) Executor {
	return &execImpl{logger: logger}
}

func (e *execImpl) Run(ctx context.Context, name string, args ...string) (string, error) {
	return e.run(ctx, "", name, args...)
}

func (e *execImpl) RunWithInput(ctx context.Context, input string, name string, args ...string) (string, error) {
	return e.run(ctx, input, name, args...)
}

func (e *execImpl) run(ctx context.Context, input string, name string, args ...string) (string, error) {
	start := time.Now()

	// Use context timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Provide stdin if needed
	if input != "" {
		cmd.Stdin = bytes.NewBufferString(input)
	}

	err := cmd.Run()
	duration := time.Since(start)

	// Log every command for audit trail
	e.logger.Info("exec",
		"cmd", name,
		"args", args,
		"duration", duration,
		"exit_code", cmd.ProcessState.ExitCode(),
	)

	if err != nil {
		return stderr.String(), fmt.Errorf("exec %s: %w (stderr: %s)", name, err, stderr.String())
	}

	return stdout.String(), nil
}
