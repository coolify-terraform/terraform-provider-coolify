package hetzner

import (
	"testing"

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

func TestHasNonDefaultHetznerSettings_AllDefaults(t *testing.T) {
	t.Parallel()
	if hasNonDefaultHetznerSettings(defaultHetznerPlan()) {
		t.Error("expected false when all fields are at their defaults")
	}
}

func TestHasNonDefaultHetznerSettings_EachField(t *testing.T) {
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
			if !hasNonDefaultHetznerSettings(m) {
				t.Errorf("expected true when %s is non-default", tc.name)
			}
		})
	}
}

func TestHasNonDefaultHetznerSettings_NullFields(t *testing.T) {
	t.Parallel()
	plan := hetznerServerResourceModel{}
	if hasNonDefaultHetznerSettings(plan) {
		t.Error("expected false when all fields are null/zero-value")
	}
}
