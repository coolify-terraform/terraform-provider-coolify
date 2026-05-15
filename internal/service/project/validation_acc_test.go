package project_test

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccProjectDataSource_InvalidUUID verifies that looking up a project
// with a malformed UUID is rejected at the plan stage by the provider's
// UUID validator, not by the API.
func TestAccProjectDataSource_InvalidUUID(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
data "coolify_project" "bad" {
  uuid = "not-a-valid-uuid"
}
`,
				ExpectError: acctest.UUIDValidationError(),
			},
		},
	})
}

// TestAccProjectDataSource_NonexistentUUID verifies that looking up a
// project with a valid-format but nonexistent UUID returns a clear error.
func TestAccProjectDataSource_NonexistentUUID(t *testing.T) {
	t.Parallel()
	acctest.AccTestSkipIfNoTFAcc(t)
	acctest.TestAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigProviderBlock() + `
data "coolify_project" "gone" {
  uuid = "00000000-0000-4000-8000-000000000000"
}
`,
				ExpectError: acctest.NotFoundError(),
			},
		},
	})
}
