#!/usr/bin/env bash
# new-resource.sh — scaffold a new Terraform resource for the Coolify provider.
#
# Usage: ./scripts/new-resource.sh <resource_name>
# Example: ./scripts/new-resource.sh webhook
#
# Creates:
#   internal/service/<name>/resource.go
#   internal/service/<name>/data_source.go
#   internal/service/<name>/resource_test.go
#   internal/service/<name>/data_source_test.go
#   internal/client/<name>s.go
#   examples/resources/coolify_<name>/resource.tf
#   examples/resources/coolify_<name>/import.sh
#   examples/data-sources/coolify_<name>/data-source.tf
#
# Then prints a checklist of remaining manual steps.
set -euo pipefail

if [ $# -ne 1 ]; then
    echo "Usage: $0 <resource_name>"
    echo "  resource_name: lowercase, underscore-separated (e.g. webhook, private_key)"
    exit 1
fi

NAME="$1"

# Derive naming variants
# snake_case -> PascalCase
pascal() {
    echo "$1" | sed -E 's/(^|_)([a-z])/\U\2/g'
}
PASCAL=$(pascal "$NAME")
LOWER="$NAME"
# For the Go package, use the name without underscores
PKG=$(echo "$NAME" | tr -d '_')

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODULE="github.com/SebTardifLabs/terraform-provider-coolify"

SVC_DIR="$REPO_ROOT/internal/service/$PKG"
CLIENT_FILE="$REPO_ROOT/internal/client/${NAME}s.go"
EXAMPLE_RES_DIR="$REPO_ROOT/examples/resources/coolify_$NAME"
EXAMPLE_DS_DIR="$REPO_ROOT/examples/data-sources/coolify_$NAME"

# Check nothing already exists
for path in "$SVC_DIR" "$CLIENT_FILE"; do
    if [ -e "$path" ]; then
        echo "Error: $path already exists. Aborting."
        exit 1
    fi
done

mkdir -p "$SVC_DIR" "$EXAMPLE_RES_DIR" "$EXAMPLE_DS_DIR"

# --- Client stubs ---
cat > "$CLIENT_FILE" << GOEOF
package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type ${PASCAL} struct {
	UUID string \`json:"uuid"\`
	Name string \`json:"name"\`
	// TODO: add fields from the Coolify API model.
}

type Create${PASCAL}Input struct {
	Name string \`json:"name"\`
	// TODO: add create input fields.
}

type Update${PASCAL}Input struct {
	Name *string \`json:"name,omitempty"\`
	// TODO: add update input fields.
}

func (c *Client) List${PASCAL}s(ctx context.Context) ([]${PASCAL}, error) {
	var r []${PASCAL}
	if err := c.do(ctx, http.MethodGet, "/api/v1/${NAME}s", nil, &r); err != nil {
		return nil, fmt.Errorf("listing ${NAME}s: %w", err)
	}
	return r, nil
}

func (c *Client) Get${PASCAL}(ctx context.Context, uuid string) (*${PASCAL}, error) {
	var r ${PASCAL}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/${NAME}s/%s", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting ${NAME} %s: %w", uuid, err)
	}
	return &r, nil
}

func (c *Client) Create${PASCAL}(ctx context.Context, input Create${PASCAL}Input) (*${PASCAL}, error) {
	var r ${PASCAL}
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/${NAME}s", input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating ${NAME}: %w", err)
	}
	return &r, nil
}

func (c *Client) Update${PASCAL}(ctx context.Context, uuid string, input Update${PASCAL}Input) (*${PASCAL}, error) {
	var r ${PASCAL}
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/${NAME}s/%s", url.PathEscape(uuid)), input, &r); err != nil {
		return nil, fmt.Errorf("updating ${NAME} %s: %w", uuid, err)
	}
	return &r, nil
}

func (c *Client) Delete${PASCAL}(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/${NAME}s/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting ${NAME} %s: %w", uuid, err)
	}
	return nil
}
GOEOF

# --- Resource ---
cat > "$SVC_DIR/resource.go" << GOEOF
package ${PKG}

import (
	"context"
	"fmt"

	"${MODULE}/internal/client"
	"${MODULE}/internal/flex"
	"${MODULE}/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*${LOWER}Resource)(nil)
	_ resource.ResourceWithImportState = (*${LOWER}Resource)(nil)
	_ resource.ResourceWithConfigure   = (*${LOWER}Resource)(nil)
)

type ${LOWER}Resource struct {
	client *client.Client
}

type ${LOWER}ResourceModel struct {
	UUID types.String \`tfsdk:"uuid"\`
	Name types.String \`tfsdk:"name"\`
	// TODO: add model fields matching the schema.
}

func NewResource() resource.Resource {
	return &${LOWER}Resource{}
}

func (r *${LOWER}Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_${NAME}"
}

func (r *${LOWER}Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify ${NAME}.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name.",
				Required:            true,
			},
			// TODO: add schema attributes.
		},
	}
}

func (r *${LOWER}Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *${LOWER}Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ${LOWER}ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_${NAME}"})

	input := client.Create${PASCAL}Input{
		Name: plan.Name.ValueString(),
	}

	created, err := r.client.Create${PASCAL}(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ${NAME}", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.client.Get${PASCAL}(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading ${NAME} after create", err.Error())
		return
	}
	flatten${PASCAL}(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *${LOWER}Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ${LOWER}ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_${NAME}", "uuid": state.UUID.ValueString()})

	got, err := r.client.Get${PASCAL}(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ${NAME}", fmt.Sprintf("${NAME} %s: %s", state.UUID.ValueString(), err))
		return
	}

	flatten${PASCAL}(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *${LOWER}Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ${LOWER}ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state ${LOWER}ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_${NAME}", "uuid": state.UUID.ValueString()})

	input := client.Update${PASCAL}Input{
		Name: flex.StringIfChanged(plan.Name, state.Name),
	}

	_, err := r.client.Update${PASCAL}(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ${NAME}", err.Error())
		return
	}

	got, err := r.client.Get${PASCAL}(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading ${NAME} after update", err.Error())
		return
	}
	flatten${PASCAL}(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *${LOWER}Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ${LOWER}ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_${NAME}", "uuid": state.UUID.ValueString()})

	if err := r.client.Delete${PASCAL}(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting ${NAME}", fmt.Sprintf("${NAME} %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *${LOWER}Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
}

func flatten${PASCAL}(src *client.${PASCAL}, m *${LOWER}ResourceModel) {
	m.UUID = types.StringValue(src.UUID)
	m.Name = types.StringValue(src.Name)
	// TODO: flatten additional fields.
}
GOEOF

# --- Data source ---
cat > "$SVC_DIR/data_source.go" << GOEOF
package ${PKG}

import (
	"context"
	"fmt"

	"${MODULE}/internal/client"
	"${MODULE}/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*${LOWER}DataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*${LOWER}DataSource)(nil)
)

type ${LOWER}DataSource struct {
	client *client.Client
}

type ${LOWER}DataSourceModel struct {
	UUID types.String \`tfsdk:"uuid"\`
	Name types.String \`tfsdk:"name"\`
	// TODO: add data source model fields.
}

func NewDataSource() datasource.DataSource {
	return &${LOWER}DataSource{}
}

func (d *${LOWER}DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_${NAME}"
}

func (d *${LOWER}DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to read a Coolify ${NAME}.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name.",
				Computed:            true,
			},
			// TODO: add data source schema attributes.
		},
	}
}

func (d *${LOWER}DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *${LOWER}DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ${LOWER}DataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := d.client.Get${PASCAL}(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading ${NAME}", fmt.Sprintf("${NAME} %s: %s", config.UUID.ValueString(), err))
		return
	}

	config.UUID = types.StringValue(got.UUID)
	config.Name = types.StringValue(got.Name)
	// TODO: flatten additional data source fields.

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
GOEOF

# --- Resource test ---
cat > "$SVC_DIR/resource_test.go" << 'GOEOF'
package PKGPLACEHOLDER_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"MODULEPLACEHOLDER/internal/acctest"
	"MODULEPLACEHOLDER/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPASCALPLACEHOLDERResource_Create(t *testing.T) {
	t.Parallel()
	obj := client.PASCALPLACEHOLDER{
		UUID: "test-uuid-0001",
		Name: "test-NAMEPLACEHOLDER",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/NAMEPLACEHOLDERs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(obj)
	})
	mux.HandleFunc("GET /api/v1/NAMEPLACEHOLDERs/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != obj.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(obj)
	})
	mux.HandleFunc("DELETE /api/v1/NAMEPLACEHOLDERs/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != obj.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_NAMEPLACEHOLDER", "/api/v1/NAMEPLACEHOLDERs/"),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestResourceConfig(srv.URL, "coolify_NAMEPLACEHOLDER", "test", `
					name = "test-NAMEPLACEHOLDER"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_NAMEPLACEHOLDER.test", "uuid", "test-uuid-0001"),
					resource.TestCheckResourceAttr("coolify_NAMEPLACEHOLDER.test", "name", "test-NAMEPLACEHOLDER"),
				),
			},
		},
	})
}
GOEOF

# Replace placeholders in test file (can't use heredoc variables in single-quoted heredoc)
sed -i "s/PKGPLACEHOLDER/${PKG}/g" "$SVC_DIR/resource_test.go"
sed -i "s|MODULEPLACEHOLDER|${MODULE}|g" "$SVC_DIR/resource_test.go"
sed -i "s/PASCALPLACEHOLDER/${PASCAL}/g" "$SVC_DIR/resource_test.go"
sed -i "s/NAMEPLACEHOLDER/${NAME}/g" "$SVC_DIR/resource_test.go"

# --- Data source test ---
cat > "$SVC_DIR/data_source_test.go" << 'GOEOF'
package PKGPLACEHOLDER_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"MODULEPLACEHOLDER/internal/acctest"
	"MODULEPLACEHOLDER/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPASCALPLACEHOLDERDataSource_Read(t *testing.T) {
	t.Parallel()
	obj := client.PASCALPLACEHOLDER{
		UUID: "ds-uuid-0001",
		Name: "ds-NAMEPLACEHOLDER",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/NAMEPLACEHOLDERs/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != obj.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(obj)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.TestDataSourceConfig(srv.URL, "coolify_NAMEPLACEHOLDER", "test", `
					uuid = "ds-uuid-0001"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_NAMEPLACEHOLDER.test", "name", "ds-NAMEPLACEHOLDER"),
				),
			},
		},
	})
}
GOEOF

sed -i "s/PKGPLACEHOLDER/${PKG}/g" "$SVC_DIR/data_source_test.go"
sed -i "s|MODULEPLACEHOLDER|${MODULE}|g" "$SVC_DIR/data_source_test.go"
sed -i "s/PASCALPLACEHOLDER/${PASCAL}/g" "$SVC_DIR/data_source_test.go"
sed -i "s/NAMEPLACEHOLDER/${NAME}/g" "$SVC_DIR/data_source_test.go"

# --- Example files ---
cat > "$EXAMPLE_RES_DIR/resource.tf" << TFEOF
resource "coolify_${NAME}" "example" {
  name = "my-${NAME}"
}
TFEOF

cat > "$EXAMPLE_RES_DIR/import.sh" << SHEOF
terraform import coolify_${NAME}.example <uuid>
SHEOF

cat > "$EXAMPLE_DS_DIR/data-source.tf" << TFEOF
data "coolify_${NAME}" "example" {
  uuid = "<uuid>"
}
TFEOF

# Format generated Go files so CI lint never fails on heredoc drift.
gofmt -w "$SVC_DIR"/*.go "$CLIENT_FILE"

echo ""
echo "Scaffolded coolify_${NAME} resource:"
echo "  $SVC_DIR/resource.go"
echo "  $SVC_DIR/data_source.go"
echo "  $SVC_DIR/resource_test.go"
echo "  $SVC_DIR/data_source_test.go"
echo "  $CLIENT_FILE"
echo "  $EXAMPLE_RES_DIR/resource.tf"
echo "  $EXAMPLE_RES_DIR/import.sh"
echo "  $EXAMPLE_DS_DIR/data-source.tf"
echo ""
echo "Remaining manual steps:"
echo "  1. Register in internal/provider/provider.go:"
echo "     - Add ${PKG}.NewResource() to Resources()"
echo "     - Add ${PKG}.NewDataSource() to DataSources()"
echo "  2. Fill in TODO placeholders in generated files"
echo "  3. Add acceptance tests in ${SVC_DIR}/resource_acc_test.go"
echo "  4. Add API endpoints to coveredEndpoints() in internal/spectest/"
echo "  5. Run: make api-coverage"
echo "  6. Run: make docs"
echo "  7. Run: make ci"
echo "  8. Update resource/data source counts in AGENTS.md and README.md"