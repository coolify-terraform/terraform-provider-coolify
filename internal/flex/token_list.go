package flex

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ReadFilteredTokenList lists cloud-token-scoped items and applies HCL filter
// blocks. listFn is the client method; the unused Client pointer that used to
// sit next to it is intentionally omitted.
func ReadFilteredTokenList[T any](
	ctx context.Context,
	tokenUUID string,
	filters []filter.Config,
	dataSourceType string,
	listErrorSummary string,
	resp *datasource.ReadResponse,
	listFn func(context.Context, string) ([]T, error),
	accessor func(T, string) (string, bool),
) ([]T, bool) {
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": dataSourceType})

	items, err := listFn(ctx, tokenUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			listErrorSummary,
			fmt.Sprintf("cloud_provider_token_uuid=%s: %s", tokenUUID, err),
		)
		return nil, false
	}

	return filter.Apply(ctx, items, filters, accessor), true
}
