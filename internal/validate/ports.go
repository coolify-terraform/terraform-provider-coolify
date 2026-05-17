package validate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// PortMappings returns a string validator that checks host:container port
// pairs are comma-separated and each port is between 1 and 65535.
func PortMappings() validator.String {
	return portMappingsValidator{}
}

type portMappingsValidator struct{}

func (v portMappingsValidator) Description(_ context.Context) string {
	return "must be comma-separated host:container port pairs with ports between 1 and 65535 (e.g. \"8080:5432\")"
}

func (v portMappingsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v portMappingsValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Port Mapping",
			"port mappings must not be empty; omit the attribute instead of setting it to an empty string")
		return
	}

	for _, pair := range strings.Split(value, ",") {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid Port Mapping",
				fmt.Sprintf("expected host:container format, got %q", pair))
			return
		}
		for _, p := range parts {
			port, err := strconv.Atoi(p)
			if err != nil || port < 1 || port > 65535 {
				resp.Diagnostics.AddAttributeError(req.Path, "Invalid Port Number",
					fmt.Sprintf("port %q must be an integer between 1 and 65535", p))
				return
			}
		}
	}
}
