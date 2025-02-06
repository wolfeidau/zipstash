package tokens

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.opentelemetry.io/otel/attribute"

	"github.com/wolfeidau/zipstash/pkg/trace"
)

type buildkite struct {
}

func newBuildkite() *buildkite {
	return &buildkite{}
}

func (b *buildkite) getIDToken(ctx context.Context, audience string) (string, error) {
	ctx, span := trace.Start(ctx, "buildkite.getIDToken")
	defer span.End()

	span.SetAttributes(attribute.String("audience", audience))

	result, err := runCommand(ctx, "", "buildkite-agent", "oidc", "request-token", "--audience", audience, "--claim", "organization_id")
	if err != nil {
		return "", fmt.Errorf("failed to get OIDC token: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("error getting OIDC token: %s", result.Stderr)
	}

	return strings.TrimSpace(result.Stdout), nil
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func runCommand(ctx context.Context, workingDir string, args ...string) (*CommandResult, error) {
	_, span := trace.Start(ctx, "runCommand")
	defer span.End()

	span.SetAttributes(attribute.StringSlice("command", args))

	cr := &CommandResult{}

	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = os.Environ() // inherit the environment

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	err := cmd.Run()
	if err != nil {
		span.RecordError(err)
		if exitError, ok := err.(*exec.ExitError); ok {
			cr.ExitCode = exitError.ExitCode()
		} else {
			return nil, err
		}
	}

	cr.Stdout = stdout.String()
	cr.Stderr = stderr.String()

	return cr, nil
}
