package client

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	for _, arg := range os.Args[1:] {
		const flag = "-coolify-header-helper="
		if strings.HasPrefix(arg, flag) {
			headerHelper(strings.TrimPrefix(arg, flag))
			return
		}
	}
	os.Exit(m.Run())
}

func headerHelper(mode string) {
	switch mode {
	case "ok":
		_, _ = os.Stdout.WriteString("secret-token\n")
	case "space":
		for _, arg := range os.Args {
			if arg == "one arg" {
				_, _ = os.Stdout.WriteString("ok")
				return
			}
		}
		_, _ = os.Stderr.WriteString("missing argument")
		os.Exit(2)
	case "fail":
		_, _ = os.Stdout.WriteString("secret-token")
		_, _ = os.Stderr.WriteString("boom secret-token")
		os.Exit(1)
	case "empty":
		return
	case "multi":
		_, _ = os.Stdout.WriteString("line\nmore")
	case "block":
		time.Sleep(time.Hour)
	default:
		os.Exit(2)
	}
}

func helperCommand(mode string, extra ...string) HeaderCommand {
	argv := []string{os.Args[0], "-coolify-header-helper=" + mode}
	argv = append(argv, extra...)
	return HeaderCommand{
		Argv:    argv,
		Timeout: 2 * time.Second,
	}
}

func TestExecuteHeaderCommand(t *testing.T) {
	ctx := context.Background()

	t.Run("prefix", func(t *testing.T) {
		cmd := helperCommand("ok")
		cmd.Prefix = "Bearer "
		got, err := ExecuteHeaderCommand(ctx, cmd)
		require.NoError(t, err)
		assert.Equal(t, "Bearer secret-token", got)
	})

	t.Run("space stays one argument", func(t *testing.T) {
		_, err := ExecuteHeaderCommand(ctx, helperCommand("space", "one arg"))
		require.NoError(t, err)
	})

	t.Run("non-zero exit omits stdout", func(t *testing.T) {
		_, err := ExecuteHeaderCommand(ctx, helperCommand("fail"))
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "secret-token")
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("empty stdout", func(t *testing.T) {
		_, err := ExecuteHeaderCommand(ctx, helperCommand("empty"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no output")
	})

	t.Run("multiline rejected", func(t *testing.T) {
		_, err := ExecuteHeaderCommand(ctx, helperCommand("multi"))
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "line")
		assert.Contains(t, err.Error(), "not a valid header value")
	})

	t.Run("timeout", func(t *testing.T) {
		cmd := helperCommand("block")
		cmd.Timeout = 100 * time.Millisecond
		start := time.Now()
		_, err := ExecuteHeaderCommand(ctx, cmd)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
		assert.Less(t, time.Since(start), 5*time.Second)
	})
}

func TestExecuteHeaderCommand_DoesNotLogArgv(t *testing.T) {
	secretArg := "super-secret-arg"
	_, err := ExecuteHeaderCommand(context.Background(), helperCommand("fail", secretArg))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), secretArg)
	assert.Contains(t, err.Error(), "failed")
}
