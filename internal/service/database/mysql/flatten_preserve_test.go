package mysql

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenDatabase_EmptyRootPasswordPreservesState(t *testing.T) {
	t.Parallel()
	m := mysqlDatabaseResourceModel{
		MysqlRootPassword: types.StringValue("rootsecret"),
		MysqlPassword:     types.StringValue("appsecret"),
	}
	flattenDatabase(&client.Database{}, &m)
	if m.MysqlRootPassword.ValueString() != "rootsecret" {
		t.Fatalf("root password = %q", m.MysqlRootPassword.ValueString())
	}
	if m.MysqlPassword.ValueString() != "appsecret" {
		t.Fatalf("password = %q", m.MysqlPassword.ValueString())
	}
}
