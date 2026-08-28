package application

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenNoindexDomains_PreservesConfigOrder(t *testing.T) {
	t.Parallel()
	configured := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("https://zebra.example.com"),
		types.StringValue("https://alpha.example.com"),
	})
	flattenNoindexDomains([]string{"https://alpha.example.com", "https://zebra.example.com"}, &configured)
	got := stringListFromTypes(configured)
	if len(got) != 2 || got[0] != "https://zebra.example.com" || got[1] != "https://alpha.example.com" {
		t.Fatalf("got %v, want configured order zebra then alpha", got)
	}
}

func TestFlattenNoindexDomains_PreservesEquivalentCase(t *testing.T) {
	t.Parallel()
	configured := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("https://Staging.Example.com"),
	})
	flattenNoindexDomains([]string{"https://staging.example.com"}, &configured)
	got := stringListFromTypes(configured)
	if len(got) != 1 || got[0] != "https://Staging.Example.com" {
		t.Fatalf("got %v, want configured casing", got)
	}
}

func TestFlattenNoindexDomains_UsesAPIWhenSetChanges(t *testing.T) {
	t.Parallel()
	configured := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("https://old.example.com"),
	})
	flattenNoindexDomains([]string{"https://new.example.com"}, &configured)
	got := stringListFromTypes(configured)
	if len(got) != 1 || got[0] != "https://new.example.com" {
		t.Fatalf("got %v, want API value", got)
	}
}

func TestFlattenNoindexDomains_ImportUsesAPIOrder(t *testing.T) {
	t.Parallel()
	dst := types.ListNull(types.StringType)
	flattenNoindexDomains([]string{"https://b.example.com", "https://a.example.com"}, &dst)
	got := stringListFromTypes(dst)
	if len(got) != 2 || got[0] != "https://b.example.com" || got[1] != "https://a.example.com" {
		t.Fatalf("got %v, want API order on import", got)
	}
}

func TestFlattenNoindexDomains_EmptyAPIClearsConfigured(t *testing.T) {
	t.Parallel()
	configured := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("https://gone.example.com"),
	})
	flattenNoindexDomains(nil, &configured)
	if configured.IsNull() {
		t.Fatal("configured empty API should be empty list, not null")
	}
	if got := stringListFromTypes(configured); len(got) != 0 {
		t.Fatalf("got %v, want empty list", got)
	}
}
