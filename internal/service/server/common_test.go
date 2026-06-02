package server

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newTestPtrs() (ServerCommonPtrs, *testModel) {
	m := &testModel{}
	return ServerCommonPtrs{
		UUID: &m.UUID, Name: &m.Name, Description: &m.Description,
		IP: &m.IP, User: &m.User, PrivateKeyUUID: &m.PrivateKeyUUID,
		Port: &m.Port, ConcurrentBuilds: &m.ConcurrentBuilds, DynamicTimeout: &m.DynamicTimeout,
		DeploymentQueueLimit:                 &m.DeploymentQueueLimit,
		ConnectionTimeout:                    &m.ConnectionTimeout,
		ServerDiskUsageNotificationThreshold: &m.ServerDiskUsageNotificationThreshold,
		ServerDiskUsageCheckFrequency:        &m.ServerDiskUsageCheckFrequency,
		IsBuildServer:                        &m.IsBuildServer, IsReachable: &m.IsReachable, IsUsable: &m.IsUsable,
		WildcardDomain: &m.WildcardDomain, IsCloudFlareTunnel: &m.IsCloudFlareTunnel,
		ServerTimezone: &m.ServerTimezone, IsMetricsEnabled: &m.IsMetricsEnabled,
		IsTerminalEnabled: &m.IsTerminalEnabled, IsSentinelEnabled: &m.IsSentinelEnabled,
		SentinelMetricsHistoryDays: &m.SentinelMetricsHistoryDays, SentinelMetricsRefreshRateSeconds: &m.SentinelMetricsRefreshRateSeconds,
		SentinelPushIntervalSeconds: &m.SentinelPushIntervalSeconds,
		DockerCleanupFrequency:      &m.DockerCleanupFrequency, DockerCleanupThreshold: &m.DockerCleanupThreshold,
		ForceDockerCleanup: &m.ForceDockerCleanup, DeleteUnusedVolumes: &m.DeleteUnusedVolumes,
		DeleteUnusedNetworks: &m.DeleteUnusedNetworks, GenerateExactLabels: &m.GenerateExactLabels,
	}, m
}

type testModel struct {
	UUID, Name, Description, IP, User, PrivateKeyUUID types.String
	Port, ConcurrentBuilds, DynamicTimeout            types.Int64
	DeploymentQueueLimit, ConnectionTimeout           types.Int64
	ServerDiskUsageNotificationThreshold              types.Int64
	ServerDiskUsageCheckFrequency                     types.String
	IsBuildServer, IsReachable, IsUsable              types.Bool
	// Extended settings
	WildcardDomain                    types.String
	IsCloudFlareTunnel                types.Bool
	ServerTimezone                    types.String
	IsMetricsEnabled                  types.Bool
	IsTerminalEnabled                 types.Bool
	IsSentinelEnabled                 types.Bool
	SentinelMetricsHistoryDays        types.Int64
	SentinelMetricsRefreshRateSeconds types.Int64
	SentinelPushIntervalSeconds       types.Int64
	DockerCleanupFrequency            types.String
	DockerCleanupThreshold            types.Int64
	ForceDockerCleanup                types.Bool
	DeleteUnusedVolumes               types.Bool
	DeleteUnusedNetworks              types.Bool
	GenerateExactLabels               types.Bool
}

var readOnlyExtendedSettingNames = []string{
	"wildcard_domain",
	"is_cloudflare_tunnel",
	"server_timezone",
	"is_metrics_enabled",
	"is_terminal_enabled",
	"is_sentinel_enabled",
	"sentinel_metrics_history_days",
	"sentinel_metrics_refresh_rate_seconds",
	"sentinel_push_interval_seconds",
	"docker_cleanup_frequency",
	"docker_cleanup_threshold",
	"force_docker_cleanup",
	"delete_unused_volumes",
	"delete_unused_networks",
	"generate_exact_labels",
}

var expectedWritableServerUpdateKeys = []string{
	"concurrent_builds",
	"connection_timeout",
	"deployment_queue_limit",
	"description",
	"dynamic_timeout",
	"ip",
	"is_build_server",
	"name",
	"port",
	"private_key_uuid",
	"server_disk_usage_check_frequency",
	"server_disk_usage_notification_threshold",
	"user",
}

func TestFlattenServerCommon_FullServer(t *testing.T) {
	t.Parallel()
	ptrs, m := newTestPtrs()
	srv := &client.Server{
		UUID: "test-uuid", Name: "my-server", Description: "desc",
		IP: "10.0.0.1", Port: 2222, User: "deploy",
		PrivateKeyUUID: "key-uuid",
		IsBuildServer:  true, IsReachable: true, IsUsable: true,
		Settings: &client.ServerSettings{
			ConcurrentBuilds:                     4,
			DynamicTimeout:                       7200,
			DeploymentQueueLimit:                 10,
			ConnectionTimeout:                    30,
			ServerDiskUsageNotificationThreshold: 90,
			ServerDiskUsageCheckFrequency:        "0 * * * *",
			WildcardDomain:                       "example.com",
			IsCloudFlareTunnel:                   true,
			ServerTimezone:                       "America/New_York",
			IsMetricsEnabled:                     true,
			IsTerminalEnabled:                    true,
			IsSentinelEnabled:                    true,
			SentinelMetricsHistoryDays:           14,
			SentinelMetricsRefreshRateSeconds:    30,
			SentinelPushIntervalSeconds:          120,
			DockerCleanupFrequency:               "0 3 * * *",
			DockerCleanupThreshold:               85,
			ForceDockerCleanup:                   true,
			DeleteUnusedVolumes:                  true,
			DeleteUnusedNetworks:                 true,
			GenerateExactLabels:                  true,
		},
	}

	FlattenServerCommon(srv, ptrs)

	checks := []struct{ name, got, want string }{
		{"UUID", m.UUID.ValueString(), "test-uuid"},
		{"Name", m.Name.ValueString(), "my-server"},
		{"Description", m.Description.ValueString(), "desc"},
		{"IP", m.IP.ValueString(), "10.0.0.1"},
		{"User", m.User.ValueString(), "deploy"},
		{"PrivateKeyUUID", m.PrivateKeyUUID.ValueString(), "key-uuid"},
		{"WildcardDomain", m.WildcardDomain.ValueString(), "example.com"},
		{"ServerTimezone", m.ServerTimezone.ValueString(), "America/New_York"},
		{"DockerCleanupFrequency", m.DockerCleanupFrequency.ValueString(), "0 3 * * *"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if m.Port.ValueInt64() != 2222 {
		t.Errorf("Port = %d, want 2222", m.Port.ValueInt64())
	}
	if !m.IsBuildServer.ValueBool() {
		t.Error("IsBuildServer = false, want true")
	}
	if m.ConcurrentBuilds.ValueInt64() != 4 {
		t.Errorf("ConcurrentBuilds = %d, want 4", m.ConcurrentBuilds.ValueInt64())
	}
	if m.DynamicTimeout.ValueInt64() != 7200 {
		t.Errorf("DynamicTimeout = %d, want 7200", m.DynamicTimeout.ValueInt64())
	}
	if m.DeploymentQueueLimit.ValueInt64() != 10 {
		t.Errorf("DeploymentQueueLimit = %d, want 10", m.DeploymentQueueLimit.ValueInt64())
	}
	if m.ConnectionTimeout.ValueInt64() != 30 {
		t.Errorf("ConnectionTimeout = %d, want 30", m.ConnectionTimeout.ValueInt64())
	}
	if m.ServerDiskUsageNotificationThreshold.ValueInt64() != 90 {
		t.Errorf("DiskUsageThreshold = %d, want 90", m.ServerDiskUsageNotificationThreshold.ValueInt64())
	}
	if m.ServerDiskUsageCheckFrequency.ValueString() != "0 * * * *" {
		t.Errorf("DiskUsageFrequency = %q, want %q", m.ServerDiskUsageCheckFrequency.ValueString(), "0 * * * *")
	}
	// Extended settings assertions
	if !m.IsCloudFlareTunnel.ValueBool() {
		t.Error("IsCloudFlareTunnel = false, want true")
	}
	if !m.IsMetricsEnabled.ValueBool() {
		t.Error("IsMetricsEnabled = false, want true")
	}
	if !m.IsTerminalEnabled.ValueBool() {
		t.Error("IsTerminalEnabled = false, want true")
	}
	if !m.IsSentinelEnabled.ValueBool() {
		t.Error("IsSentinelEnabled = false, want true")
	}
	if m.SentinelMetricsHistoryDays.ValueInt64() != 14 {
		t.Errorf("SentinelMetricsHistoryDays = %d, want 14", m.SentinelMetricsHistoryDays.ValueInt64())
	}
	if m.SentinelMetricsRefreshRateSeconds.ValueInt64() != 30 {
		t.Errorf("SentinelMetricsRefreshRateSeconds = %d, want 30", m.SentinelMetricsRefreshRateSeconds.ValueInt64())
	}
	if m.SentinelPushIntervalSeconds.ValueInt64() != 120 {
		t.Errorf("SentinelPushIntervalSeconds = %d, want 120", m.SentinelPushIntervalSeconds.ValueInt64())
	}
	if m.DockerCleanupThreshold.ValueInt64() != 85 {
		t.Errorf("DockerCleanupThreshold = %d, want 85", m.DockerCleanupThreshold.ValueInt64())
	}
	if !m.ForceDockerCleanup.ValueBool() {
		t.Error("ForceDockerCleanup = false, want true")
	}
	if !m.DeleteUnusedVolumes.ValueBool() {
		t.Error("DeleteUnusedVolumes = false, want true")
	}
	if !m.DeleteUnusedNetworks.ValueBool() {
		t.Error("DeleteUnusedNetworks = false, want true")
	}
	if !m.GenerateExactLabels.ValueBool() {
		t.Error("GenerateExactLabels = false, want true")
	}
}

func TestFlattenServerCommon_NilSettings(t *testing.T) {
	t.Parallel()
	ptrs, m := newTestPtrs()
	// Pre-populate settings fields to verify they are NOT overwritten when Settings is nil.
	m.ConcurrentBuilds = types.Int64Value(99)
	m.DynamicTimeout = types.Int64Value(99)

	srv := &client.Server{
		UUID: "uuid", Name: "n", IP: "1.2.3.4", Port: 22, User: "root",
		Settings: nil,
	}

	FlattenServerCommon(srv, ptrs)

	if m.ConcurrentBuilds.ValueInt64() != 99 {
		t.Errorf("ConcurrentBuilds changed to %d, want 99 (preserved)", m.ConcurrentBuilds.ValueInt64())
	}
	if m.DynamicTimeout.ValueInt64() != 99 {
		t.Errorf("DynamicTimeout changed to %d, want 99 (preserved)", m.DynamicTimeout.ValueInt64())
	}
}

func TestFlattenServerCommon_ZeroConnectionTimeoutDefaultsTo10(t *testing.T) {
	t.Parallel()
	ptrs, m := newTestPtrs()

	srv := &client.Server{
		UUID: "uuid", Name: "n", IP: "1.2.3.4", Port: 22, User: "root",
		Settings: &client.ServerSettings{
			ConcurrentBuilds:                     2,
			DynamicTimeout:                       3600,
			DeploymentQueueLimit:                 25,
			ConnectionTimeout:                    0,
			ServerDiskUsageNotificationThreshold: 80,
		},
	}

	FlattenServerCommon(srv, ptrs)

	if m.ConnectionTimeout.ValueInt64() != 10 {
		t.Errorf("ConnectionTimeout = %d, want 10", m.ConnectionTimeout.ValueInt64())
	}
}

func TestFlattenServerCommon_EmptyPrivateKeyUUID(t *testing.T) {
	t.Parallel()
	ptrs, m := newTestPtrs()
	m.PrivateKeyUUID = types.StringValue("original-key")

	srv := &client.Server{
		UUID: "uuid", Name: "n", IP: "1.2.3.4", Port: 22, User: "root",
		PrivateKeyUUID: "", // API omits this on GET
	}

	FlattenServerCommon(srv, ptrs)

	if m.PrivateKeyUUID.ValueString() != "original-key" {
		t.Errorf("PrivateKeyUUID = %q, want %q (preserved)", m.PrivateKeyUUID.ValueString(), "original-key")
	}
}

func TestBuildServerUpdateInput_NoChanges(t *testing.T) {
	t.Parallel()
	plan, _ := newTestPtrs()
	state, _ := newTestPtrs()

	// Set identical values on both.
	v := types.StringValue("same")
	*plan.Name = v
	*state.Name = v
	*plan.Description = v
	*state.Description = v
	*plan.IP = v
	*state.IP = v
	*plan.User = v
	*state.User = v
	*plan.PrivateKeyUUID = v
	*state.PrivateKeyUUID = v
	*plan.ServerDiskUsageCheckFrequency = v
	*state.ServerDiskUsageCheckFrequency = v

	p := types.Int64Value(22)
	*plan.Port = p
	*state.Port = p
	*plan.ConcurrentBuilds = p
	*state.ConcurrentBuilds = p
	*plan.DynamicTimeout = p
	*state.DynamicTimeout = p
	*plan.DeploymentQueueLimit = p
	*state.DeploymentQueueLimit = p
	*plan.ConnectionTimeout = p
	*state.ConnectionTimeout = p
	*plan.ServerDiskUsageNotificationThreshold = p
	*state.ServerDiskUsageNotificationThreshold = p

	b := types.BoolValue(false)
	*plan.IsBuildServer = b
	*state.IsBuildServer = b

	input := BuildServerUpdateInput(plan, state)

	if input.Name != nil {
		t.Errorf("Name should be nil when unchanged, got %v", *input.Name)
	}
	if input.Port != nil {
		t.Errorf("Port should be nil when unchanged, got %v", *input.Port)
	}
	if input.IsBuildServer != nil {
		t.Errorf("IsBuildServer should be nil when unchanged, got %v", *input.IsBuildServer)
	}
}

func TestBuildServerUpdateInput_PartialChange(t *testing.T) {
	t.Parallel()
	plan, _ := newTestPtrs()
	state, _ := newTestPtrs()

	*plan.Name = types.StringValue("new-name")
	*state.Name = types.StringValue("old-name")
	// Keep port identical.
	*plan.Port = types.Int64Value(22)
	*state.Port = types.Int64Value(22)
	// All other fields null.

	input := BuildServerUpdateInput(plan, state)

	if input.Name == nil || *input.Name != "new-name" {
		t.Errorf("Name should be %q, got %v", "new-name", input.Name)
	}
	if input.Port != nil {
		t.Errorf("Port should be nil when unchanged, got %v", *input.Port)
	}
}

func TestCommonServerAttrs_ExtendedSettingsAreReadOnly(t *testing.T) {
	t.Parallel()

	attrs := CommonServerAttrs(context.Background(), map[string]schema.Attribute{})
	for _, name := range readOnlyExtendedSettingNames {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("missing attribute %q", name)
		}
		switch a := attr.(type) {
		case schema.StringAttribute:
			if !a.Computed || a.Optional || a.Required {
				t.Errorf("%s should be computed-only, got Optional=%v Required=%v Computed=%v", name, a.Optional, a.Required, a.Computed)
			}
		case schema.BoolAttribute:
			if !a.Computed || a.Optional || a.Required {
				t.Errorf("%s should be computed-only, got Optional=%v Required=%v Computed=%v", name, a.Optional, a.Required, a.Computed)
			}
		case schema.Int64Attribute:
			if !a.Computed || a.Optional || a.Required {
				t.Errorf("%s should be computed-only, got Optional=%v Required=%v Computed=%v", name, a.Optional, a.Required, a.Computed)
			}
		default:
			t.Fatalf("unexpected attribute type %T for %s", attr, name)
		}
	}
}

func TestUpdateServerInput_PublicPatchSurfaceMatchesExpectedKeys(t *testing.T) {
	t.Parallel()

	updateType := reflect.TypeOf(client.UpdateServerInput{})
	actualKeys := make([]string, 0, updateType.NumField())
	for i := 0; i < updateType.NumField(); i++ {
		key, _, _ := strings.Cut(updateType.Field(i).Tag.Get("json"), ",")
		if key == "" || key == "-" {
			continue
		}
		actualKeys = append(actualKeys, key)
	}
	actualKeys = sortedStrings(actualKeys)
	if !reflect.DeepEqual(actualKeys, expectedWritableServerUpdateKeys) {
		t.Fatalf("UpdateServerInput PATCH keys = %v, want %v", actualKeys, expectedWritableServerUpdateKeys)
	}
}

func TestBuildServerUpdateInput_AllFieldsChanged(t *testing.T) {
	t.Parallel()
	plan, _ := newTestPtrs()
	state, _ := newTestPtrs()

	// String fields: plan != state.
	*plan.Name = types.StringValue("new-name")
	*state.Name = types.StringValue("old-name")
	*plan.Description = types.StringValue("new-desc")
	*state.Description = types.StringValue("old-desc")
	*plan.IP = types.StringValue("10.0.0.2")
	*state.IP = types.StringValue("10.0.0.1")
	*plan.User = types.StringValue("deploy")
	*state.User = types.StringValue("root")
	*plan.PrivateKeyUUID = types.StringValue("new-key")
	*state.PrivateKeyUUID = types.StringValue("old-key")
	*plan.ServerDiskUsageCheckFrequency = types.StringValue("*/10 * * * *")
	*state.ServerDiskUsageCheckFrequency = types.StringValue("*/5 * * * *")

	// Int64 fields: plan != state.
	*plan.Port = types.Int64Value(2222)
	*state.Port = types.Int64Value(22)
	*plan.ConcurrentBuilds = types.Int64Value(8)
	*state.ConcurrentBuilds = types.Int64Value(2)
	*plan.DynamicTimeout = types.Int64Value(7200)
	*state.DynamicTimeout = types.Int64Value(3600)
	*plan.DeploymentQueueLimit = types.Int64Value(50)
	*state.DeploymentQueueLimit = types.Int64Value(25)
	*plan.ConnectionTimeout = types.Int64Value(30)
	*state.ConnectionTimeout = types.Int64Value(10)
	*plan.ServerDiskUsageNotificationThreshold = types.Int64Value(95)
	*state.ServerDiskUsageNotificationThreshold = types.Int64Value(80)

	// Bool field: plan != state.
	*plan.IsBuildServer = types.BoolValue(true)
	*state.IsBuildServer = types.BoolValue(false)

	// Unsupported extended settings still appear on reads, but the update
	// contract must never write them back.
	*plan.WildcardDomain = types.StringValue("new.example.com")
	*state.WildcardDomain = types.StringValue("old.example.com")
	*plan.IsCloudFlareTunnel = types.BoolValue(true)
	*state.IsCloudFlareTunnel = types.BoolValue(false)
	*plan.ServerTimezone = types.StringValue("America/New_York")
	*state.ServerTimezone = types.StringValue("UTC")
	*plan.IsMetricsEnabled = types.BoolValue(true)
	*state.IsMetricsEnabled = types.BoolValue(false)
	*plan.IsTerminalEnabled = types.BoolValue(true)
	*state.IsTerminalEnabled = types.BoolValue(false)
	*plan.IsSentinelEnabled = types.BoolValue(true)
	*state.IsSentinelEnabled = types.BoolValue(false)
	*plan.SentinelMetricsHistoryDays = types.Int64Value(7)
	*state.SentinelMetricsHistoryDays = types.Int64Value(0)
	*plan.SentinelMetricsRefreshRateSeconds = types.Int64Value(10)
	*state.SentinelMetricsRefreshRateSeconds = types.Int64Value(0)
	*plan.SentinelPushIntervalSeconds = types.Int64Value(60)
	*state.SentinelPushIntervalSeconds = types.Int64Value(0)
	*plan.DockerCleanupFrequency = types.StringValue("0 3 * * *")
	*state.DockerCleanupFrequency = types.StringValue("0 0 * * *")
	*plan.DockerCleanupThreshold = types.Int64Value(90)
	*state.DockerCleanupThreshold = types.Int64Value(80)
	*plan.ForceDockerCleanup = types.BoolValue(true)
	*state.ForceDockerCleanup = types.BoolValue(false)
	*plan.DeleteUnusedVolumes = types.BoolValue(true)
	*state.DeleteUnusedVolumes = types.BoolValue(false)
	*plan.DeleteUnusedNetworks = types.BoolValue(true)
	*state.DeleteUnusedNetworks = types.BoolValue(false)
	*plan.GenerateExactLabels = types.BoolValue(true)
	*state.GenerateExactLabels = types.BoolValue(false)

	input := BuildServerUpdateInput(plan, state)

	// Verify string fields.
	stringChecks := []struct {
		name, want string
		got        *string
	}{
		{"Name", "new-name", input.Name},
		{"Description", "new-desc", input.Description},
		{"IP", "10.0.0.2", input.IP},
		{"User", "deploy", input.User},
		{"PrivateKeyUUID", "new-key", input.PrivateKeyUUID},
		{"ServerDiskUsageCheckFrequency", "*/10 * * * *", input.ServerDiskUsageCheckFrequency},
	}
	for _, c := range stringChecks {
		if c.got == nil {
			t.Errorf("%s should be non-nil", c.name)
		} else if *c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, *c.got, c.want)
		}
	}

	// Verify int fields.
	intChecks := []struct {
		name string
		want int
		got  *int
	}{
		{"Port", 2222, input.Port},
		{"ConcurrentBuilds", 8, input.ConcurrentBuilds},
		{"DynamicTimeout", 7200, input.DynamicTimeout},
		{"DeploymentQueueLimit", 50, input.DeploymentQueueLimit},
		{"ConnectionTimeout", 30, input.ConnectionTimeout},
		{"ServerDiskUsageNotificationThreshold", 95, input.ServerDiskUsageNotificationThreshold},
	}
	for _, c := range intChecks {
		if c.got == nil {
			t.Errorf("%s should be non-nil", c.name)
		} else if *c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, *c.got, c.want)
		}
	}

	// Verify bool field.
	if input.IsBuildServer == nil {
		t.Error("IsBuildServer should be non-nil")
	} else if *input.IsBuildServer != true {
		t.Errorf("IsBuildServer = %v, want true", *input.IsBuildServer)
	}

	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal update input: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal update input: %v", err)
	}
	for _, key := range readOnlyExtendedSettingNames {
		if _, ok := body[key]; ok {
			t.Errorf("unexpected unsupported PATCH field %q in body %s", key, payload)
		}
	}

	actualKeys := make([]string, 0, len(body))
	for key := range body {
		actualKeys = append(actualKeys, key)
	}
	actualKeys = sortedStrings(actualKeys)
	if !reflect.DeepEqual(actualKeys, expectedWritableServerUpdateKeys) {
		t.Fatalf("BuildServerUpdateInput PATCH keys = %v, want %v", actualKeys, expectedWritableServerUpdateKeys)
	}
}

func sortedStrings(values []string) []string {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return copyValues
}

func TestBuildPostCreateSettingsInput_AllDefaults(t *testing.T) {
	t.Parallel()
	ptrs, m := newTestPtrs()
	m.ConcurrentBuilds = types.Int64Value(2)
	m.DynamicTimeout = types.Int64Value(3600)
	m.DeploymentQueueLimit = types.Int64Value(25)
	m.ConnectionTimeout = types.Int64Value(10)
	m.ServerDiskUsageNotificationThreshold = types.Int64Value(80)

	input := BuildPostCreateSettingsInput(ptrs)

	if input.ConcurrentBuilds != nil {
		t.Errorf("ConcurrentBuilds should be nil at default, got %v", *input.ConcurrentBuilds)
	}
	if input.DynamicTimeout != nil {
		t.Errorf("DynamicTimeout should be nil at default, got %v", *input.DynamicTimeout)
	}
	if input.DeploymentQueueLimit != nil {
		t.Errorf("DeploymentQueueLimit should be nil at default, got %v", *input.DeploymentQueueLimit)
	}
	if input.ConnectionTimeout != nil {
		t.Errorf("ConnectionTimeout should be nil at default, got %v", *input.ConnectionTimeout)
	}
	if input.ServerDiskUsageNotificationThreshold != nil {
		t.Errorf("DiskUsageThreshold should be nil at default, got %v", *input.ServerDiskUsageNotificationThreshold)
	}
	if input.ServerDiskUsageCheckFrequency != nil {
		t.Errorf("DiskUsageFrequency should be nil at default, got %v", *input.ServerDiskUsageCheckFrequency)
	}
}

func TestBuildPostCreateSettingsInput_NonDefaults(t *testing.T) {
	t.Parallel()
	ptrs, m := newTestPtrs()
	m.ConcurrentBuilds = types.Int64Value(8)
	m.DynamicTimeout = types.Int64Value(7200)
	m.DeploymentQueueLimit = types.Int64Value(50)
	m.ConnectionTimeout = types.Int64Value(30)
	m.ServerDiskUsageNotificationThreshold = types.Int64Value(95)
	m.ServerDiskUsageCheckFrequency = types.StringValue("*/10 * * * *")

	input := BuildPostCreateSettingsInput(ptrs)

	intChecks := []struct {
		name string
		want int
		got  *int
	}{
		{"ConcurrentBuilds", 8, input.ConcurrentBuilds},
		{"DynamicTimeout", 7200, input.DynamicTimeout},
		{"DeploymentQueueLimit", 50, input.DeploymentQueueLimit},
		{"ConnectionTimeout", 30, input.ConnectionTimeout},
		{"ServerDiskUsageNotificationThreshold", 95, input.ServerDiskUsageNotificationThreshold},
	}
	for _, c := range intChecks {
		if c.got == nil {
			t.Errorf("%s should be non-nil", c.name)
		} else if *c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, *c.got, c.want)
		}
	}
	if input.ServerDiskUsageCheckFrequency == nil {
		t.Error("ServerDiskUsageCheckFrequency should be non-nil")
	} else if *input.ServerDiskUsageCheckFrequency != "*/10 * * * *" {
		t.Errorf("ServerDiskUsageCheckFrequency = %q, want %q", *input.ServerDiskUsageCheckFrequency, "*/10 * * * *")
	}

	// Verify the input does NOT include fields that belong only in Update.
	if input.Name != nil {
		t.Errorf("Name should be nil in post-create settings input, got %v", *input.Name)
	}
	if input.IP != nil {
		t.Errorf("IP should be nil in post-create settings input, got %v", *input.IP)
	}
}

func TestHasNonDefaultSettings_ViaCommonPtrs(t *testing.T) {
	t.Parallel()
	ptrs, m := newTestPtrs()

	// All defaults: should return false.
	m.ConcurrentBuilds = types.Int64Value(2)
	m.DynamicTimeout = types.Int64Value(3600)
	m.DeploymentQueueLimit = types.Int64Value(25)
	m.ConnectionTimeout = types.Int64Value(10)
	m.ServerDiskUsageNotificationThreshold = types.Int64Value(80)

	if HasNonDefaultSettings(ptrs) {
		t.Error("expected false when all fields are at their defaults")
	}

	// One non-default: should return true.
	m.ConcurrentBuilds = types.Int64Value(8)
	if !HasNonDefaultSettings(ptrs) {
		t.Error("expected true when ConcurrentBuilds is non-default")
	}
}

func TestHasNonDefaultSettings_AllDefaults(t *testing.T) {
	t.Parallel()
	plan := serverResourceModel{
		ConcurrentBuilds:                     types.Int64Value(2),
		DynamicTimeout:                       types.Int64Value(3600),
		DeploymentQueueLimit:                 types.Int64Value(25),
		ConnectionTimeout:                    types.Int64Value(10),
		ServerDiskUsageNotificationThreshold: types.Int64Value(80),
	}
	if hasNonDefaultSettings(plan) {
		t.Error("expected false when all fields are at their defaults")
	}
}

func TestHasNonDefaultSettings_EachField(t *testing.T) {
	t.Parallel()
	base := func() serverResourceModel {
		return serverResourceModel{
			ConcurrentBuilds:                     types.Int64Value(2),
			DynamicTimeout:                       types.Int64Value(3600),
			DeploymentQueueLimit:                 types.Int64Value(25),
			ConnectionTimeout:                    types.Int64Value(10),
			ServerDiskUsageNotificationThreshold: types.Int64Value(80),
		}
	}
	cases := []struct {
		name   string
		mutate func(*serverResourceModel)
	}{
		{"ConcurrentBuilds", func(m *serverResourceModel) { m.ConcurrentBuilds = types.Int64Value(8) }},
		{"DynamicTimeout", func(m *serverResourceModel) { m.DynamicTimeout = types.Int64Value(7200) }},
		{"DeploymentQueueLimit", func(m *serverResourceModel) { m.DeploymentQueueLimit = types.Int64Value(50) }},
		{"ConnectionTimeout", func(m *serverResourceModel) { m.ConnectionTimeout = types.Int64Value(30) }},
		{"ServerDiskUsageNotificationThreshold", func(m *serverResourceModel) {
			m.ServerDiskUsageNotificationThreshold = types.Int64Value(95)
		}},
		{"ServerDiskUsageCheckFrequency", func(m *serverResourceModel) {
			m.ServerDiskUsageCheckFrequency = types.StringValue("*/10 * * * *")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := base()
			tc.mutate(&m)
			if !hasNonDefaultSettings(m) {
				t.Errorf("expected true when %s is non-default", tc.name)
			}
		})
	}
}

func TestHasNonDefaultSettings_NullFields(t *testing.T) {
	t.Parallel()
	plan := serverResourceModel{}
	if hasNonDefaultSettings(plan) {
		t.Error("expected false when all fields are null/zero-value")
	}
}
