package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// HostingerDataCenter is one Hostinger VPS location.
type HostingerDataCenter struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	City      string `json:"city"`
	Continent string `json:"continent"`
}

// HostingerCatalogPrice is one billing price on a catalog item. Price is cents.
type HostingerCatalogPrice struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Currency         string `json:"currency"`
	Price            int64  `json:"price"`
	FirstPeriodPrice int64  `json:"first_period_price"`
	Period           int64  `json:"period"`
	PeriodUnit       string `json:"period_unit"`
}

// HostingerCatalogItem is a KVM VPS plan Coolify offers for purchase.
type HostingerCatalogItem struct {
	ID       string                  `json:"id"`
	Name     string                  `json:"name"`
	Category string                  `json:"category"`
	Prices   []HostingerCatalogPrice `json:"prices"`
}

// HostingerTemplate is a Hostinger OS template.
type HostingerTemplate struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Documentation string `json:"documentation"`
}

// HostingerPublicKey is an SSH public key stored in the Hostinger account.
type HostingerPublicKey struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// HostingerPostInstallScript is a script Hostinger can run after VPS setup.
type HostingerPostInstallScript struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateHostingerServerInput purchases a Hostinger VPS and registers it.
// Requires Coolify >= v4.5.0.
type CreateHostingerServerInput struct {
	CloudProviderTokenUUID string  `json:"cloud_provider_token_uuid"`
	ItemID                 string  `json:"item_id"`
	DataCenterID           int64   `json:"data_center_id"`
	TemplateID             int64   `json:"template_id"`
	Name                   string  `json:"name,omitempty"`
	PrivateKeyUUID         string  `json:"private_key_uuid"`
	EnableBackups          *bool   `json:"enable_backups,omitempty"`
	PublicKeyIDs           []int64 `json:"public_key_ids,omitempty"`
	PostInstallScriptID    *int64  `json:"post_install_script_id,omitempty"`
	InstantValidate        *bool   `json:"instant_validate,omitempty"`
}

// HostingerServerCreated is the purchase response, not a full server record.
type HostingerServerCreated struct {
	UUID                      string `json:"uuid"`
	HostingerVirtualMachineID int64  `json:"hostinger_virtual_machine_id"`
	IP                        string `json:"ip"`
	Provisioning              bool   `json:"provisioning"`
}

func hostingerPath(base, cloudProviderTokenUUID string) string {
	q := url.Values{}
	q.Set("cloud_provider_token_uuid", cloudProviderTokenUUID)
	return base + "?" + q.Encode()
}

func (c *Client) ListHostingerDataCenters(ctx context.Context, tokenUUID string) ([]HostingerDataCenter, error) {
	var out []HostingerDataCenter
	if err := c.do(ctx, http.MethodGet, hostingerPath("/api/v1/hostinger/data-centers", tokenUUID), nil, &out); err != nil {
		return nil, fmt.Errorf("listing hostinger data centers: %w", err)
	}
	return out, nil
}

func (c *Client) ListHostingerCatalog(ctx context.Context, tokenUUID string) ([]HostingerCatalogItem, error) {
	var out []HostingerCatalogItem
	if err := c.do(ctx, http.MethodGet, hostingerPath("/api/v1/hostinger/catalog", tokenUUID), nil, &out); err != nil {
		return nil, fmt.Errorf("listing hostinger catalog: %w", err)
	}
	return out, nil
}

func (c *Client) ListHostingerTemplates(ctx context.Context, tokenUUID string) ([]HostingerTemplate, error) {
	var out []HostingerTemplate
	if err := c.do(ctx, http.MethodGet, hostingerPath("/api/v1/hostinger/templates", tokenUUID), nil, &out); err != nil {
		return nil, fmt.Errorf("listing hostinger templates: %w", err)
	}
	return out, nil
}

func (c *Client) ListHostingerSSHKeys(ctx context.Context, tokenUUID string) ([]HostingerPublicKey, error) {
	var out []HostingerPublicKey
	if err := c.do(ctx, http.MethodGet, hostingerPath("/api/v1/hostinger/ssh-keys", tokenUUID), nil, &out); err != nil {
		return nil, fmt.Errorf("listing hostinger ssh keys: %w", err)
	}
	return out, nil
}

func (c *Client) ListHostingerPostInstallScripts(ctx context.Context, tokenUUID string) ([]HostingerPostInstallScript, error) {
	var out []HostingerPostInstallScript
	if err := c.do(ctx, http.MethodGet, hostingerPath("/api/v1/hostinger/post-install-scripts", tokenUUID), nil, &out); err != nil {
		return nil, fmt.Errorf("listing hostinger post-install scripts: %w", err)
	}
	return out, nil
}

func (c *Client) CreateHostingerServer(ctx context.Context, input CreateHostingerServerInput) (*HostingerServerCreated, error) {
	var out HostingerServerCreated
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/servers/hostinger", input, &out, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating hostinger server: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("creating hostinger server: API returned empty UUID")
	}
	return &out, nil
}
