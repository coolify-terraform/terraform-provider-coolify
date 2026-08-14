package flex_test

import (
	"context"
	"errors"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type tokenListItem struct {
	Name string
}

func TestReadFilteredTokenList_SuccessAndFilter(t *testing.T) {
	t.Parallel()

	resp := &datasource.ReadResponse{}
	filters := []filter.Config{{
		Name:   types.StringValue("name"),
		Values: []types.String{types.StringValue("keep")},
	}}
	items, ok := flex.ReadFilteredTokenList(
		context.Background(),
		"tok",
		filters,
		"test_ds",
		"Error listing test items",
		resp,
		func(_ context.Context, token string) ([]tokenListItem, error) {
			if token != "tok" {
				t.Fatalf("token = %q", token)
			}
			return []tokenListItem{{Name: "keep"}, {Name: "drop"}}, nil
		},
		func(item tokenListItem, field string) (string, bool) {
			if field == "name" {
				return item.Name, true
			}
			return "", false
		},
	)
	if !ok {
		t.Fatal("expected success")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if len(items) != 1 || items[0].Name != "keep" {
		t.Fatalf("items = %#v", items)
	}
}

func TestReadFilteredTokenList_ListError(t *testing.T) {
	t.Parallel()

	resp := &datasource.ReadResponse{}
	items, ok := flex.ReadFilteredTokenList(
		context.Background(),
		"missing",
		nil,
		"test_ds",
		"Error listing test items",
		resp,
		func(context.Context, string) ([]tokenListItem, error) {
			return nil, errors.New("token not found")
		},
		func(tokenListItem, string) (string, bool) { return "", false },
	)
	if ok || items != nil {
		t.Fatalf("ok=%v items=%v", ok, items)
	}
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostic")
	}
	errs := resp.Diagnostics.Errors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if got := errs[0].Summary(); got != "Error listing test items" {
		t.Fatalf("summary = %q", got)
	}
	if got := errs[0].Detail(); got != "token not found" {
		t.Fatalf("detail = %q", got)
	}
}
