package flex

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// CreateReadBackTaintedReplace is the recovery clause after Create saved
// partial state and then returned an error. Terraform taints that object,
// so the next apply destroys it and creates a new one. Refresh does not
// clear the taint.
const CreateReadBackTaintedReplace = "The partial Terraform state was saved and Terraform marked the resource tainted, so the next terraform apply replaces it by destroying the existing object and creating a new one, and terraform refresh does not clear that mark."

// CreateReadBackFailedSummary is the diagnostic summary when create succeeded
// but the immediate read-back failed.
func CreateReadBackFailedSummary(label string) string {
	return fmt.Sprintf("%s created but refresh failed", label)
}

// CreateReadBackFailedDetail is the diagnostic detail for a generic read-back
// error after create. label appears in both "created" and "Could not read" clauses.
func CreateReadBackFailedDetail(label, identifier string, err error) string {
	return fmt.Sprintf(
		"Coolify created %s %s, but the provider could not read it back: Could not read %s %s after create: %s. %s",
		label,
		identifier,
		label,
		identifier,
		err,
		CreateReadBackTaintedReplace,
	)
}

// CreateReadBackNotFoundDetail is the diagnostic detail when the immediate GET
// after create returns 404 (create/read race).
func CreateReadBackNotFoundDetail(label, identifier string) string {
	return fmt.Sprintf(
		"Coolify created %s %s, but the provider could not read it back because the API returned 404 on the immediate read-back. %s",
		label,
		identifier,
		CreateReadBackTaintedReplace,
	)
}

// AddCreateReadBackError records a Create diagnostic when Coolify created the
// resource but the provider could not refresh it. Callers must already have
// saved partial state so Terraform tracks the remote object.
//
// label is used for both summary and detail. When historical wording used
// different casing (e.g. summary "S3 storage" vs body "s3 storage"), call
// CreateReadBackFailedSummary/Detail separately and AddError yourself.
// identifier is typically the UUID returned by create.
func AddCreateReadBackError(resp *resource.CreateResponse, label, identifier string, err error) {
	resp.Diagnostics.AddError(
		CreateReadBackFailedSummary(label),
		CreateReadBackFailedDetail(label, identifier, err),
	)
}

// AddCreateReadBackNotFoundError is like AddCreateReadBackError when the
// immediate GET returns 404. Callers must already have saved partial state.
func AddCreateReadBackNotFoundError(resp *resource.CreateResponse, label, identifier string) {
	resp.Diagnostics.AddError(
		CreateReadBackFailedSummary(label),
		CreateReadBackNotFoundDetail(label, identifier),
	)
}
