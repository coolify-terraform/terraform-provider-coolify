package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	maxHeaderCommandStdout = 8192
	maxHeaderCommandStderr = 512
	headerCommandWaitDelay = 200 * time.Millisecond
)

// HeaderCommand is one provider header whose value is a command's stdout.
type HeaderCommand struct {
	Argv    []string
	Prefix  string
	Timeout time.Duration
}

// ExecuteHeaderCommand runs argv once and returns prefix plus the trimmed
// stdout. The error text names the program only, never the arguments or
// the stdout bytes.
func ExecuteHeaderCommand(ctx context.Context, cmd HeaderCommand) (string, error) {
	if len(cmd.Argv) == 0 || strings.TrimSpace(cmd.Argv[0]) == "" {
		return "", errors.New("header command is empty")
	}
	program := cmd.Argv[0]
	timeout := cmd.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	child := exec.CommandContext(runCtx, program, cmd.Argv[1:]...) // #nosec G204 -- argv is the provider block, not a shell
	child.WaitDelay = headerCommandWaitDelay
	var stdout, stderr capBuffer
	stdout.limit = maxHeaderCommandStdout
	stderr.limit = maxHeaderCommandStderr
	child.Stdout = &stdout
	child.Stderr = &stderr

	err := child.Run()
	if stdout.overflow {
		return "", fmt.Errorf("header command %s output exceeds %d bytes", program, maxHeaderCommandStdout)
	}
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("header command %s timed out after %s", program, timeout)
		}
		excerpt := scrubStderr(stderr.String(), stdout.String())
		if excerpt == "" {
			return "", fmt.Errorf("header command %s failed", program)
		}
		return "", fmt.Errorf("header command %s failed: %s", program, excerpt)
	}
	trimmed := strings.TrimRight(stdout.String(), " \t\r\n")
	if trimmed == "" {
		return "", fmt.Errorf("header command %s produced no output", program)
	}
	value := cmd.Prefix + trimmed
	if !validHeaderValue(value) {
		return "", fmt.Errorf("header command %s output is not a valid header value", program)
	}
	return value, nil
}

func scrubStderr(stderr, stdout string) string {
	stderr = strings.TrimSpace(stderr)
	if stdout != "" {
		stderr = strings.ReplaceAll(stderr, stdout, "")
		stderr = strings.TrimSpace(stderr)
	}
	return stderr
}

// capBuffer keeps at most limit bytes and reports overflow.
// Write acknowledges the full slice so a producer does not stop early.
type capBuffer struct {
	buf      bytes.Buffer
	limit    int
	overflow bool
}

func (c *capBuffer) Write(p []byte) (int, error) {
	remain := c.limit - c.buf.Len()
	if remain <= 0 {
		c.overflow = true
		return len(p), nil
	}
	if len(p) > remain {
		c.overflow = true
		_, _ = c.buf.Write(p[:remain])
		return len(p), nil
	}
	return c.buf.Write(p)
}

func (c *capBuffer) String() string { return c.buf.String() }
