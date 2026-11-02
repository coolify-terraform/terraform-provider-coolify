package provider

import (
	"context"
	"crypto/x509"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func proxyBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Outbound proxy for the machine running Terraform (a CI runner or an HCP Terraform agent). " +
			"This is not the authenticating proxy in front of Coolify; that credential belongs in `headers`. " +
			"When `url` is set it replaces `HTTP_PROXY` and `HTTPS_PROXY`. " +
			"Supported schemes are `http`, `socks5`, and `socks5h`. An `https` proxy URL is rejected because Go would apply the Coolify TLS settings to the proxy hop. " +
			"Env: `COOLIFY_PROXY_URL`, `COOLIFY_PROXY_USERNAME`, `COOLIFY_PROXY_PASSWORD`, `COOLIFY_NO_PROXY`. A set attribute wins. " +
			"`url` is required when the block is set. It is optional in the schema so an omitted block does not require it.",
		Attributes: map[string]schema.Attribute{
			"url":      schema.StringAttribute{MarkdownDescription: "Proxy address, such as `http://proxy.example:8080` or `socks5://proxy.example:1080`. Required when the block is set.", Optional: true},
			"username": schema.StringAttribute{MarkdownDescription: "Username for proxy basic auth. Prefer this over userinfo in `url`.", Optional: true},
			"password": schema.StringAttribute{MarkdownDescription: "Password for proxy basic auth.", Optional: true, Sensitive: true},
			"no_proxy": schema.StringAttribute{MarkdownDescription: "Comma-separated hosts that connect directly. An empty string bypasses nothing. When omitted, `NO_PROXY` from the environment still applies.", Optional: true},
		},
	}
}

func headerCommandBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: "Runs a command once per Terraform command and sends its stdout as one header. " +
			"Use this for a short-lived proxy credential such as `gcloud auth print-identity-token`. " +
			"The command is not a shell. The value is not refreshed during apply. " +
			"`Authorization` is rejected. On Terraform 1.10 or later, prefer an ephemeral value in `headers` when the token already exists as one.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{MarkdownDescription: "Header name. The names rejected by `headers` are rejected here.", Required: true},
				"command": schema.ListAttribute{
					MarkdownDescription: "Program and arguments. The first element is the program.",
					ElementType:         types.StringType,
					Required:            true,
					Validators:          []validator.List{listvalidator.SizeAtLeast(1)},
				},
				"prefix":  schema.StringAttribute{MarkdownDescription: "Text prepended to stdout, such as `Bearer `.", Optional: true},
				"timeout": schema.Int64Attribute{MarkdownDescription: "Seconds to wait for the command (default: 30, maximum: 300).", Optional: true, Validators: []validator.Int64{int64validator.Between(1, 300)}},
			},
		},
	}
}

func validateProviderTLS(cfg client.RetryConfig, resp *tfprovider.ConfigureResponse) bool {
	if cfg.Insecure && cfg.CACert != "" {
		resp.Diagnostics.AddWarning("CA certificate ignored",
			"Both insecure and ca_cert are set. When insecure is true, "+
				"TLS certificate verification is skipped entirely and ca_cert is not used.")
	}
	if cfg.CACert != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(cfg.CACert)) {
			resp.Diagnostics.AddError("Invalid CA Certificate",
				"The ca_cert value could not be parsed as a PEM-encoded certificate. "+
					"Check that the value is a valid PEM block starting with -----BEGIN CERTIFICATE-----.")
			return false
		}
	}
	if err := client.ValidateRetryConfig(cfg); err != nil {
		resp.Diagnostics.AddError("Invalid TLS configuration", err.Error())
		return false
	}
	return true
}

func configureConnectivity(ctx context.Context, endpoint string, config coolifyProviderModel, cfg *client.RetryConfig) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.Append(applyRequestTimeout(config, cfg)...)
	diags.Append(applyTLSServerName(config, cfg)...)
	diags.Append(applyClientCert(config, cfg)...)
	diags.Append(applyProxy(config, cfg)...)
	diags.Append(applyHeaderCommands(ctx, config, cfg)...)
	if diags.HasError() {
		return diags
	}
	diags.Append(warnConnectivity(endpoint, cfg)...)
	return diags
}

func applyRequestTimeout(config coolifyProviderModel, cfg *client.RetryConfig) diag.Diagnostics {
	var diags diag.Diagnostics
	if config.RequestTimeout.IsUnknown() {
		diags.AddAttributeError(path.Root("request_timeout"), "Unknown request timeout", "request_timeout must be known during planning.")
		return diags
	}
	if !config.RequestTimeout.IsNull() {
		cfg.RequestTimeout = time.Duration(config.RequestTimeout.ValueInt64()) * time.Second
		return nil
	}
	raw := strings.TrimSpace(os.Getenv("COOLIFY_REQUEST_TIMEOUT"))
	if raw == "" {
		return nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 300 {
		diags.AddAttributeError(path.Root("request_timeout"), "Invalid COOLIFY_REQUEST_TIMEOUT",
			"COOLIFY_REQUEST_TIMEOUT must be a whole number of seconds from 1 to 300.")
		return diags
	}
	cfg.RequestTimeout = time.Duration(n) * time.Second
	return nil
}

func applyTLSServerName(config coolifyProviderModel, cfg *client.RetryConfig) diag.Diagnostics {
	value, diags := configuredString(config.TLSServerName, "COOLIFY_TLS_SERVER_NAME", path.Root("tls_server_name"))
	if diags.HasError() {
		return diags
	}
	cfg.TLSServerName = value
	return nil
}

func applyClientCert(config coolifyProviderModel, cfg *client.RetryConfig) diag.Diagnostics {
	var diags diag.Diagnostics
	if config.ClientCert.IsUnknown() || config.ClientKey.IsUnknown() {
		diags.AddError("Unknown client certificate", "client_cert and client_key must be known during planning.")
		return diags
	}
	certSet := !config.ClientCert.IsNull()
	keySet := !config.ClientKey.IsNull()
	if certSet || keySet {
		if certSet {
			cfg.ClientCert = config.ClientCert.ValueString()
		}
		if keySet {
			cfg.ClientKey = config.ClientKey.ValueString()
		}
		return nil
	}
	cfg.ClientCert = os.Getenv("COOLIFY_CLIENT_CERT")
	cfg.ClientKey = os.Getenv("COOLIFY_CLIENT_KEY")
	return nil
}

func applyProxy(config coolifyProviderModel, cfg *client.RetryConfig) diag.Diagnostics {
	settings, diags := proxySettings(config)
	if diags.HasError() {
		return diags
	}
	fn, err := client.ProxyFunction(settings)
	if err != nil {
		diags.AddAttributeError(path.Root("proxy").AtName("url"), "Invalid proxy", err.Error())
		return diags
	}
	cfg.Proxy = fn
	return nil
}

func proxySettings(config coolifyProviderModel) (client.ProxySettings, diag.Diagnostics) {
	var diags diag.Diagnostics
	var settings client.ProxySettings
	if config.Proxy == nil {
		settings.URL = strings.TrimSpace(os.Getenv("COOLIFY_PROXY_URL"))
		settings.Username = os.Getenv("COOLIFY_PROXY_USERNAME")
		settings.Password = os.Getenv("COOLIFY_PROXY_PASSWORD")
		if _, ok := os.LookupEnv("COOLIFY_NO_PROXY"); ok {
			settings.NoProxySet = true
			settings.NoProxy = os.Getenv("COOLIFY_NO_PROXY")
		}
		return settings, nil
	}
	block := config.Proxy
	if block.URL.IsUnknown() || block.Username.IsUnknown() || block.Password.IsUnknown() || block.NoProxy.IsUnknown() {
		diags.AddAttributeError(path.Root("proxy"), "Unknown proxy", "proxy values must be known during planning.")
		return settings, diags
	}
	if block.URL.IsNull() && block.Username.IsNull() && block.Password.IsNull() && block.NoProxy.IsNull() {
		// Protocol 6 materializes an omitted single nested block as an empty
		// object. Treat that as "no proxy block".
		settings.URL = strings.TrimSpace(os.Getenv("COOLIFY_PROXY_URL"))
		settings.Username = os.Getenv("COOLIFY_PROXY_USERNAME")
		settings.Password = os.Getenv("COOLIFY_PROXY_PASSWORD")
		if _, ok := os.LookupEnv("COOLIFY_NO_PROXY"); ok {
			settings.NoProxySet = true
			settings.NoProxy = os.Getenv("COOLIFY_NO_PROXY")
		}
		return settings, nil
	}
	settings.URL = strings.TrimSpace(os.Getenv("COOLIFY_PROXY_URL"))
	if !block.URL.IsNull() {
		settings.URL = strings.TrimSpace(block.URL.ValueString())
	}
	settings.Username = os.Getenv("COOLIFY_PROXY_USERNAME")
	if !block.Username.IsNull() {
		settings.Username = block.Username.ValueString()
	}
	settings.Password = os.Getenv("COOLIFY_PROXY_PASSWORD")
	if !block.Password.IsNull() {
		settings.Password = block.Password.ValueString()
	}
	if !block.NoProxy.IsNull() {
		settings.NoProxySet = true
		settings.NoProxy = block.NoProxy.ValueString()
	} else if _, ok := os.LookupEnv("COOLIFY_NO_PROXY"); ok {
		settings.NoProxySet = true
		settings.NoProxy = os.Getenv("COOLIFY_NO_PROXY")
	}
	return settings, nil
}

func applyHeaderCommands(ctx context.Context, config coolifyProviderModel, cfg *client.RetryConfig) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(config.HeaderCommand) == 0 {
		return nil
	}
	seen := map[string]string{}
	for name := range cfg.Headers {
		seen[http.CanonicalHeaderKey(name)] = name
	}
	if cfg.Headers == nil {
		cfg.Headers = map[string]string{}
	}
	for _, block := range config.HeaderCommand {
		if block.Name.IsNull() || block.Name.IsUnknown() || block.Command.IsNull() || block.Command.IsUnknown() || block.Prefix.IsUnknown() || block.Timeout.IsUnknown() {
			diags.AddError("Unknown header command", "header_command values must be known during planning.")
			return diags
		}
		name := block.Name.ValueString()
		canonical := http.CanonicalHeaderKey(name)
		if other, ok := seen[canonical]; ok {
			diags.AddAttributeError(path.Root("header_command"), "Duplicate header",
				name+" is already set by "+other+". Set each header in one place.")
			return diags
		}
		var argv []string
		diags.Append(block.Command.ElementsAs(ctx, &argv, false)...)
		if diags.HasError() {
			return diags
		}
		probe := client.RetryConfig{Headers: map[string]string{name: "x"}, CFAccessClientID: cfg.CFAccessClientID, CFAccessClientSec: cfg.CFAccessClientSec}
		if err := probe.ValidateHeaders(); err != nil {
			diags.AddAttributeError(path.Root("header_command"), "Invalid header command", err.Error())
			return diags
		}
		timeout := 30 * time.Second
		if !block.Timeout.IsNull() {
			timeout = time.Duration(block.Timeout.ValueInt64()) * time.Second
		}
		prefix := ""
		if !block.Prefix.IsNull() {
			prefix = block.Prefix.ValueString()
		}
		value, err := client.ExecuteHeaderCommand(ctx, client.HeaderCommand{Argv: argv, Prefix: prefix, Timeout: timeout})
		if err != nil {
			diags.AddAttributeError(path.Root("header_command"), "Header command failed", err.Error())
			return diags
		}
		cfg.Headers[name] = value
		seen[canonical] = name
	}
	if err := cfg.ValidateHeaders(); err != nil {
		diags.AddAttributeError(path.Root("header_command"), "Invalid header command", err.Error())
	}
	return diags
}

func warnConnectivity(endpoint string, cfg *client.RetryConfig) diag.Diagnostics {
	var diags diag.Diagnostics
	if !strings.HasPrefix(endpoint, "http://") {
		return nil
	}
	if cfg.ClientCert != "" || cfg.ClientKey != "" {
		diags.AddWarning("Client certificate ignored",
			"The endpoint uses plain HTTP, so the client certificate is not presented. Use an https:// endpoint for mutual TLS.")
	}
	if cfg.TLSServerName != "" {
		diags.AddWarning("TLS server name ignored",
			"The endpoint uses plain HTTP, so tls_server_name has no effect. Use an https:// endpoint.")
	}
	if cfg.Proxy != nil && headerSet(cfg.Headers, "Proxy-Authorization") {
		diags.AddWarning("Proxy header overwritten on cleartext",
			"An http:// endpoint sends proxy basic auth on the same request as the Coolify call, which replaces a Proxy-Authorization header. Use https:// so that header stays on the tunneled request.")
	}
	return diags
}

func headerSet(headers map[string]string, name string) bool {
	want := http.CanonicalHeaderKey(name)
	for got := range headers {
		if http.CanonicalHeaderKey(got) == want {
			return true
		}
	}
	return false
}

func configuredString(attr types.String, envKey string, attrPath path.Path) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if attr.IsUnknown() {
		diags.AddAttributeError(attrPath, "Unknown value", attrPath.String()+" must be known during planning.")
		return "", diags
	}
	if !attr.IsNull() {
		return attr.ValueString(), nil
	}
	return os.Getenv(envKey), nil
}
