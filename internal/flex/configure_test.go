package flex_test

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestConfigureClient(t *testing.T) {
	t.Parallel()

	t.Run("nil provider data", func(t *testing.T) {
		var diags diag.Diagnostics
		got := flex.ConfigureClient(resource.ConfigureRequest{ProviderData: nil}, &diags)
		if got != nil {
			t.Fatal("expected nil client for nil provider data")
		}
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}
	})

	t.Run("correct type", func(t *testing.T) {
		c := &client.Client{}
		var diags diag.Diagnostics
		got := flex.ConfigureClient(resource.ConfigureRequest{ProviderData: c}, &diags)
		if got != c {
			t.Fatal("expected returned client to be the same pointer")
		}
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		var diags diag.Diagnostics
		got := flex.ConfigureClient(resource.ConfigureRequest{ProviderData: "not-a-client"}, &diags)
		if got != nil {
			t.Fatal("expected nil client for wrong type")
		}
		if !diags.HasError() {
			t.Fatal("expected error diagnostic for wrong type")
		}
	})
}

func TestConfigureDataSourceClient(t *testing.T) {
	t.Parallel()

	t.Run("nil provider data", func(t *testing.T) {
		var diags diag.Diagnostics
		got := flex.ConfigureDataSourceClient(datasource.ConfigureRequest{ProviderData: nil}, &diags)
		if got != nil {
			t.Fatal("expected nil client for nil provider data")
		}
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}
	})

	t.Run("correct type", func(t *testing.T) {
		c := &client.Client{}
		var diags diag.Diagnostics
		got := flex.ConfigureDataSourceClient(datasource.ConfigureRequest{ProviderData: c}, &diags)
		if got != c {
			t.Fatal("expected returned client to be the same pointer")
		}
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		var diags diag.Diagnostics
		got := flex.ConfigureDataSourceClient(datasource.ConfigureRequest{ProviderData: "not-a-client"}, &diags)
		if got != nil {
			t.Fatal("expected nil client for wrong type")
		}
		if !diags.HasError() {
			t.Fatal("expected error diagnostic for wrong type")
		}
	})
}
