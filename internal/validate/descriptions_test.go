package validate_test

import (
	"context"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/stretchr/testify/assert"
)

func TestValidatorDescriptions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	domains := validate.Domains()
	assert.NotEmpty(t, domains.Description(ctx))
	assert.Equal(t, domains.Description(ctx), domains.MarkdownDescription(ctx))

	composeDomains := validate.DockerComposeDomains()
	assert.NotEmpty(t, composeDomains.Description(ctx))
	assert.Equal(t, composeDomains.Description(ctx), composeDomains.MarkdownDescription(ctx))

	ports := validate.PortMappings()
	assert.NotEmpty(t, ports.Description(ctx))
	assert.Equal(t, ports.Description(ctx), ports.MarkdownDescription(ctx))

	shell := validate.NoShellMetachars()
	assert.NotEmpty(t, shell.Description(ctx))
	assert.Equal(t, shell.Description(ctx), shell.MarkdownDescription(ctx))
}
