The weekly contract-freshness check detected field or route changes in the upstream Coolify source code.

## After updating the pin

1. Run `make contract-extract` (or the versioned extract you use) and commit the pin (`testdata/contracts/coolify-v4.json` and versioned files as needed).
2. Run `make contract-check` and `go test ./internal/spectest/ -run 'ContractCoverage|WriteCoverage|SchemaCoverage' -count=1`.
3. For every new fillable or allow-list field:
   - Implement client + Terraform schema, **or**
   - Mark **deferred** with an open issue (`#N`) in the skip taxonomy / schema registry, **or**
   - Mark **internal** / **n/a** only if truly non-user-facing (FK, computed, wrong parent, superseded name).
4. Do **not** use comments like "internal flag" for public API fields (for example Coolify `$allowedFields` on create/update).
5. Contract pipeline references:
   - Ignore taxonomy: #622
   - `allowed_fields` vs client write input: #623
   - Schema coverage registry: #621
   - Deferred field backlog: #626
