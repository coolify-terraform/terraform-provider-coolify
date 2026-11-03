package integrationtoken

import (
	"context"
	"encoding/json"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateIntegrationToken(cfg integrationTokenModel, diags *diag.Diagnostics) {
	if diags == nil || cfg.Provider.IsNull() || cfg.Provider.IsUnknown() {
		return
	}
	provider := cfg.Provider.ValueString()
	cloudflare := provider == "cloudflare"
	if !cloudflare {
		rejectCloudflareOnly(cfg, diags)
	}
	if cloudflare {
		rejectMetadata(cfg, diags)
	}
	if provider == "doppler" && !cfg.Token.IsNull() && !cfg.Token.IsUnknown() && !dopplerTokenPattern.MatchString(cfg.Token.ValueString()) {
		diags.AddAttributeError(path.Root("token"), "Invalid Doppler token", "Doppler integration tokens must start with dp.st. or dp.sa.")
	}
	if provider == "infisical" || provider == "vault" {
		requireBaseURL(cfg, diags)
	}
	if provider == "infisical" && (cfg.MetadataClientID.IsNull() || cfg.MetadataClientID.IsUnknown() || cfg.MetadataClientID.ValueString() == "") {
		diags.AddAttributeError(path.Root("metadata_client_id"), "Missing Infisical client id", "Infisical integration tokens require metadata_client_id.")
	}
	warnTunnelWithoutDNS(cfg, diags)
}

func rejectCloudflareOnly(cfg integrationTokenModel, diags *diag.Diagnostics) {
	if !cfg.AutomaticDNS.IsNull() && !cfg.AutomaticDNS.IsUnknown() {
		diags.AddAttributeError(path.Root("automatic_dns"), "automatic_dns is Cloudflare-only", "Only Cloudflare integration tokens accept automatic_dns.")
	}
	if !cfg.ProxiedDNS.IsNull() && !cfg.ProxiedDNS.IsUnknown() {
		diags.AddAttributeError(path.Root("proxied_dns"), "proxied_dns is Cloudflare-only", "Only Cloudflare integration tokens accept proxied_dns.")
	}
}

func rejectMetadata(cfg integrationTokenModel, diags *diag.Diagnostics) {
	if stringSet(cfg.MetadataBaseURL) {
		diags.AddAttributeError(path.Root("metadata_base_url"), "metadata is not allowed for Cloudflare", "Cloudflare integration tokens reject metadata.")
	}
	if stringSet(cfg.MetadataClientID) {
		diags.AddAttributeError(path.Root("metadata_client_id"), "metadata is not allowed for Cloudflare", "Cloudflare integration tokens reject metadata.")
	}
	if stringSet(cfg.MetadataNamespace) {
		diags.AddAttributeError(path.Root("metadata_namespace"), "metadata is not allowed for Cloudflare", "Cloudflare integration tokens reject metadata.")
	}
	if !cfg.MetadataTimeout.IsNull() && !cfg.MetadataTimeout.IsUnknown() {
		diags.AddAttributeError(path.Root("metadata_timeout"), "metadata is not allowed for Cloudflare", "Cloudflare integration tokens reject metadata.")
	}
}

func requireBaseURL(cfg integrationTokenModel, diags *diag.Diagnostics) {
	if cfg.MetadataBaseURL.IsNull() || cfg.MetadataBaseURL.IsUnknown() || cfg.MetadataBaseURL.ValueString() == "" {
		diags.AddAttributeError(path.Root("metadata_base_url"), "Missing secret-manager URL", "Infisical and Vault integration tokens require metadata_base_url.")
	}
}

func warnTunnelWithoutDNS(cfg integrationTokenModel, diags *diag.Diagnostics) {
	if cfg.Capabilities.IsNull() || cfg.Capabilities.IsUnknown() {
		return
	}
	var caps []string
	d := cfg.Capabilities.ElementsAs(context.Background(), &caps, false)
	if d.HasError() {
		return
	}
	hasTunnel, hasDNS := false, false
	for _, cap := range caps {
		if cap == "tunnel" {
			hasTunnel = true
		}
		if cap == "dns" {
			hasDNS = true
		}
		if cfg.Provider.ValueString() != "cloudflare" && cap != "secrets" {
			diags.AddAttributeError(path.Root("capabilities"), "Invalid capability", "Doppler, Infisical, and Vault integration tokens only accept the secrets capability.")
		}
		if cfg.Provider.ValueString() == "cloudflare" && cap == "secrets" {
			diags.AddAttributeError(path.Root("capabilities"), "Invalid capability", "Cloudflare integration tokens accept dns and tunnel, not secrets.")
		}
	}
	if hasTunnel && !hasDNS {
		diags.AddAttributeError(path.Root("capabilities"), "Tunnel needs DNS", "Cloudflare Tunnel also needs the dns capability.")
	}
}

func stringSet(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}

func createInput(ctx context.Context, plan integrationTokenModel) client.CreateIntegrationTokenInput {
	in := client.CreateIntegrationTokenInput{
		Provider: plan.Provider.ValueString(),
		Name:     plan.Name.ValueString(),
		Token:    plan.Token.ValueString(),
	}
	if caps := stringList(ctx, plan.Capabilities); caps != nil {
		in.Capabilities = caps
	}
	in.AutomaticDNS = boolPtr(plan.AutomaticDNS)
	in.ProxiedDNS = boolPtr(plan.ProxiedDNS)
	in.Metadata = metadataPtr(plan)
	return in
}

func updateInput(ctx context.Context, plan, state integrationTokenModel) client.UpdateIntegrationTokenInput {
	var in client.UpdateIntegrationTokenInput
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		in.Name = &name
	}
	metaChanged := !plan.MetadataBaseURL.Equal(state.MetadataBaseURL) || !plan.MetadataClientID.Equal(state.MetadataClientID) || !plan.MetadataNamespace.Equal(state.MetadataNamespace) || !plan.MetadataTimeout.Equal(state.MetadataTimeout)
	capsChanged := !plan.Capabilities.Equal(state.Capabilities)
	tokenChanged := !plan.Token.Equal(state.Token)
	if tokenChanged || metaChanged || capsChanged {
		token := plan.Token.ValueString()
		in.Token = &token
	}
	if capsChanged {
		caps := stringList(ctx, plan.Capabilities)
		if caps == nil {
			empty := []string{}
			caps = empty
		}
		in.Capabilities = &caps
	}
	if !plan.AutomaticDNS.Equal(state.AutomaticDNS) {
		in.AutomaticDNS = boolPtr(plan.AutomaticDNS)
	}
	if !plan.ProxiedDNS.Equal(state.ProxiedDNS) {
		in.ProxiedDNS = boolPtr(plan.ProxiedDNS)
	}
	if metaChanged {
		in.Metadata = metadataUpdateJSON(plan, state)
	}
	return in
}

// metadataUpdateJSON is the PATCH metadata object. Unchanged keys are
// omitted so Coolify's merge keeps them. A cleared string is "" and a
// cleared timeout is null: the API drops those empty values and removes
// the stored key. Required base_url and client_id are rejected earlier
// when the config clears them.
func metadataUpdateJSON(plan, state integrationTokenModel) json.RawMessage {
	body := map[string]any{}
	putString := func(key string, next, prev types.String) {
		if next.Equal(prev) {
			return
		}
		if !stringSet(next) {
			body[key] = ""
			return
		}
		body[key] = next.ValueString()
	}
	putString("base_url", plan.MetadataBaseURL, state.MetadataBaseURL)
	putString("client_id", plan.MetadataClientID, state.MetadataClientID)
	putString("namespace", plan.MetadataNamespace, state.MetadataNamespace)
	if !plan.MetadataTimeout.Equal(state.MetadataTimeout) {
		if plan.MetadataTimeout.IsNull() || plan.MetadataTimeout.IsUnknown() {
			body["timeout"] = nil
		} else {
			body["timeout"] = plan.MetadataTimeout.ValueInt64()
		}
	}
	if len(body) == 0 {
		return nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil
	}
	return raw
}

func metadataPtr(plan integrationTokenModel) *client.IntegrationTokenMetadata {
	if !stringSet(plan.MetadataBaseURL) && !stringSet(plan.MetadataClientID) && !stringSet(plan.MetadataNamespace) && (plan.MetadataTimeout.IsNull() || plan.MetadataTimeout.IsUnknown()) {
		return nil
	}
	meta := &client.IntegrationTokenMetadata{}
	if stringSet(plan.MetadataBaseURL) {
		meta.BaseURL = plan.MetadataBaseURL.ValueString()
	}
	if stringSet(plan.MetadataClientID) {
		meta.ClientID = plan.MetadataClientID.ValueString()
	}
	if stringSet(plan.MetadataNamespace) {
		meta.Namespace = plan.MetadataNamespace.ValueString()
	}
	if !plan.MetadataTimeout.IsNull() && !plan.MetadataTimeout.IsUnknown() {
		timeout := plan.MetadataTimeout.ValueInt64()
		meta.Timeout = &timeout
	}
	return meta
}

func flattenIntegrationToken(got *client.IntegrationToken, m *integrationTokenModel) {
	if got == nil || m == nil {
		return
	}
	m.UUID = types.StringValue(got.UUID)
	if got.Name != "" {
		m.Name = types.StringValue(got.Name)
	}
	if got.Provider != "" {
		m.Provider = types.StringValue(got.Provider)
	}
	if got.Capabilities != nil {
		if !sameStringSet(m.Capabilities, got.Capabilities) {
			m.Capabilities = stringListValue(got.Capabilities)
		}
	} else if m.Capabilities.IsUnknown() {
		m.Capabilities = types.ListNull(types.StringType)
	}
	if got.AutomaticDNS != nil {
		m.AutomaticDNS = types.BoolValue(*got.AutomaticDNS)
	} else if m.AutomaticDNS.IsUnknown() {
		m.AutomaticDNS = types.BoolNull()
	}
	if got.ProxiedDNS != nil {
		m.ProxiedDNS = types.BoolValue(*got.ProxiedDNS)
	} else if m.ProxiedDNS.IsUnknown() {
		m.ProxiedDNS = types.BoolNull()
	}
	if got.DNSZonesCount != nil {
		m.DNSZonesCount = types.Int64Value(*got.DNSZonesCount)
	} else if m.DNSZonesCount.IsUnknown() {
		m.DNSZonesCount = types.Int64Null()
	}
	if got.ZonesSyncedAt != nil && *got.ZonesSyncedAt != "" {
		m.ZonesSyncedAt = types.StringValue(*got.ZonesSyncedAt)
	} else if m.ZonesSyncedAt.IsUnknown() {
		m.ZonesSyncedAt = types.StringNull()
	}
	if got.CreatedAt != "" {
		m.CreatedAt = types.StringValue(got.CreatedAt)
	} else if m.CreatedAt.IsUnknown() {
		m.CreatedAt = types.StringNull()
	}
	if got.UpdatedAt != "" {
		m.UpdatedAt = types.StringValue(got.UpdatedAt)
	} else if m.UpdatedAt.IsUnknown() {
		m.UpdatedAt = types.StringNull()
	}
}

func boolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func stringList(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var out []string
	if d := list.ElementsAs(ctx, &out, false); d.HasError() {
		return nil
	}
	return out
}

func sameStringSet(list types.List, api []string) bool {
	if list.IsNull() || list.IsUnknown() {
		return false
	}
	got := stringList(context.Background(), list)
	if len(got) != len(api) {
		return false
	}
	counts := map[string]int{}
	for _, v := range got {
		counts[v]++
	}
	for _, v := range api {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}

func stringListValue(values []string) types.List {
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elems)
}
