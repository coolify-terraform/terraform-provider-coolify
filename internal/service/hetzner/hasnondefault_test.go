package hetzner

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/server"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func defaultHetznerPlan() hetznerServerResourceModel {
	return hetznerServerResourceModel{
		Description:                          types.StringValue(""),
		Port:                                 types.Int64Value(22),
		User:                                 types.StringValue("root"),
		IsBuildServer:                        types.BoolValue(false),
		ConcurrentBuilds:                     types.Int64Value(2),
		DynamicTimeout:                       types.Int64Value(3600),
		DeploymentQueueLimit:                 types.Int64Value(25),
		ConnectionTimeout:                    types.Int64Value(10),
		ServerDiskUsageNotificationThreshold: types.Int64Value(80),
		ServerDiskUsageCheckFrequency:        types.StringValue(""),
	}
}

func TestInt64IDsFromCSV(t *testing.T) {
	t.Parallel()

	got, err := int64IDsFromCSV("12345, 67890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != 12345 || got[1] != 67890 {
		t.Fatalf("got %v, want [12345 67890]", got)
	}

	empty, err := int64IDsFromCSV("  ,  ")
	if err != nil {
		t.Fatalf("empty csv error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty csv = %v, want empty", empty)
	}

	if _, err := int64IDsFromCSV("38,abc"); err == nil {
		t.Fatal("expected error for non-integer token")
	}
}

func TestHasNonDefaultCloudProviderSettings_ViaHetznerModel(t *testing.T) {
	t.Parallel()
	m := defaultHetznerPlan()
	if server.HasNonDefaultCloudProviderSettings(m.commonPtrs()) {
		t.Error("expected false when all fields are at their defaults")
	}
}

func TestHasNonDefaultCloudProviderSettings_EachHetznerField(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*hetznerServerResourceModel)
	}{
		{"Description", func(m *hetznerServerResourceModel) { m.Description = types.StringValue("custom") }},
		{"Port", func(m *hetznerServerResourceModel) { m.Port = types.Int64Value(2222) }},
		{"User", func(m *hetznerServerResourceModel) { m.User = types.StringValue("deploy") }},
		{"IsBuildServer", func(m *hetznerServerResourceModel) { m.IsBuildServer = types.BoolValue(true) }},
		{"ConcurrentBuilds", func(m *hetznerServerResourceModel) { m.ConcurrentBuilds = types.Int64Value(8) }},
		{"DynamicTimeout", func(m *hetznerServerResourceModel) { m.DynamicTimeout = types.Int64Value(7200) }},
		{"DeploymentQueueLimit", func(m *hetznerServerResourceModel) { m.DeploymentQueueLimit = types.Int64Value(50) }},
		{"ConnectionTimeout", func(m *hetznerServerResourceModel) { m.ConnectionTimeout = types.Int64Value(30) }},
		{"ServerDiskUsageNotificationThreshold", func(m *hetznerServerResourceModel) {
			m.ServerDiskUsageNotificationThreshold = types.Int64Value(95)
		}},
		{"ServerDiskUsageCheckFrequency", func(m *hetznerServerResourceModel) {
			m.ServerDiskUsageCheckFrequency = types.StringValue("*/10 * * * *")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := defaultHetznerPlan()
			tc.mutate(&m)
			if !server.HasNonDefaultCloudProviderSettings(m.commonPtrs()) {
				t.Errorf("expected true when %s is non-default", tc.name)
			}
		})
	}
}

func TestHasNonDefaultCloudProviderSettings_NullHetznerFields(t *testing.T) {
	t.Parallel()
	plan := hetznerServerResourceModel{}
	if server.HasNonDefaultCloudProviderSettings((&plan).commonPtrs()) {
		t.Error("expected false when all fields are null/zero-value")
	}
}
