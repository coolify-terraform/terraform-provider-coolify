package acctest

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestAccPreCheck validates that required environment variables are set.
func TestAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("COOLIFY_ENDPOINT"); v == "" {
		t.Fatal("COOLIFY_ENDPOINT must be set for acceptance tests")
	}
	if v := os.Getenv("COOLIFY_TOKEN"); v == "" {
		t.Fatal("COOLIFY_TOKEN must be set for acceptance tests")
	}
}

// RandomWithPrefix generates a random string with the given prefix.
func RandomWithPrefix(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, rand.Intn(999999))
}

// TestProtoV6ProviderFactories returns provider factories for unit tests.
func TestProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"coolify": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// ConfigProviderBlock returns a Terraform HCL provider configuration block.
func ConfigProviderBlock() string {
	return fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token     = %q
}
`, os.Getenv("COOLIFY_ENDPOINT"), os.Getenv("COOLIFY_TOKEN"))
}
