package validate

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// shellMetachars matches characters that could enable shell injection
// when the value is interpolated into a command line.
var shellMetachars = regexp.MustCompile("[;|&`$(){}]")

// shellSafeCommand matches Coolify ValidationPatterns::SHELL_SAFE_COMMAND_PATTERN.
// It allows && and || and quoted args; it rejects ; | $ ` newlines and similar.
var shellSafeCommand = regexp.MustCompile("^(?:[ \\t]+|&&|\\|\\||\"[^\"$`\\\\\\n\\r]*\"|'[^'\\n\\r]*'|[a-zA-Z0-9._\\-/=:@,+\\[\\]{}#%^~*?!]+)+$")

// dockerTarget matches Coolify ValidationPatterns::DOCKER_TARGET_PATTERN.
var dockerTarget = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// NoShellMetachars returns a string validator that rejects values
// containing shell metacharacters (; | & ` $ ( ) { }).
func NoShellMetachars() validator.String {
	return noShellMetacharsValidator{}
}

// ShellSafeCommand returns a string validator matching Coolify's
// shell-safe command rules (docker compose custom start/build).
func ShellSafeCommand() validator.String {
	return shellSafeCommandValidator{}
}

// DockerTarget returns a string validator matching Coolify's
// Docker multi-stage target name rules.
func DockerTarget() validator.String {
	return dockerTargetValidator{}
}

type noShellMetacharsValidator struct{}

func (v noShellMetacharsValidator) Description(_ context.Context) string {
	return "must not contain shell metacharacters (; | & ` $ ( ) { })"
}

func (v noShellMetacharsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v noShellMetacharsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if loc := shellMetachars.FindStringIndex(value); loc != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Shell Metacharacter Detected",
			v.Description(ctx))
	}
}

type shellSafeCommandValidator struct{}

func (v shellSafeCommandValidator) Description(_ context.Context) string {
	return "must be a Coolify shell-safe command (&& and || allowed; no ; | $ ` newlines)"
}

func (v shellSafeCommandValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v shellSafeCommandValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}
	if !shellSafeCommand.MatchString(value) {
		resp.Diagnostics.AddAttributeError(req.Path, "Unsafe Command Detected",
			v.Description(ctx))
	}
}

type dockerTargetValidator struct{}

func (v dockerTargetValidator) Description(_ context.Context) string {
	return "must be a Docker multi-stage target name (letter or digit, then letters, digits, dots, hyphens, or underscores)"
}

func (v dockerTargetValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v dockerTargetValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}
	if !dockerTarget.MatchString(value) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Docker Target",
			v.Description(ctx))
	}
}
