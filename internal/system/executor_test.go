package system

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// testExecutor creates an Executor with a discarded logger.
func testExecutor() Executor {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewExecutor(logger)
}

func TestRun_Echo(t *testing.T) {
	exec := testExecutor()
	out, err := exec.Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Errorf("got %q, want %q", out, "hello")
	}
}

func TestRun_FailingCommand(t *testing.T) {
	exec := testExecutor()
	_, err := exec.Run(context.Background(), "false")
	if err == nil {
		t.Fatal("expected error from 'false' command")
	}
}

func TestRun_Timeout(t *testing.T) {
	exec := testExecutor()

	// 1ms timeout with a sleep 10 command should fail
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := exec.Run(ctx, "sleep", "10")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestRunWithInput_Cat(t *testing.T) {
	exec := testExecutor()
	out, err := exec.RunWithInput(context.Background(), "hello", "cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello" {
		t.Errorf("got %q, want %q", out, "hello")
	}
}

func TestRun_DefaultTimeout(t *testing.T) {
	exec := testExecutor()

	// No deadline on context — should apply DefaultTimeout (30s) and succeed
	out, err := exec.Run(context.Background(), "echo", "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "test" {
		t.Errorf("got %q", out)
	}
}
