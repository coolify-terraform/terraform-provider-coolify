package application

import (
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// #358: runtimeFieldsChanged
// ---------------------------------------------------------------------------

func TestRuntimeFieldsChanged_Identical(t *testing.T) {
	t.Parallel()
	name := types.StringValue("app")
	desc := types.StringValue("desc")
	dom := types.StringValue("https://example.com")
	plan := commonAppFields{Name: &name, Description: &desc, Domains: &dom}
	state := commonAppFields{Name: &name, Description: &desc, Domains: &dom}
	if runtimeFieldsChanged(plan, state) {
		t.Error("expected false for identical plan and state")
	}
}

func TestRuntimeFieldsChanged_StringChange(t *testing.T) {
	t.Parallel()
	planName := types.StringValue("new-app")
	stateName := types.StringValue("old-app")
	plan := commonAppFields{Name: &planName}
	state := commonAppFields{Name: &stateName}
	if !runtimeFieldsChanged(plan, state) {
		t.Error("expected true when Name differs")
	}
}

func TestRuntimeFieldsChanged_BoolChange(t *testing.T) {
	t.Parallel()
	planBool := types.BoolValue(true)
	stateBool := types.BoolValue(false)
	plan := commonAppFields{IsForceHTTPSEnabled: &planBool}
	state := commonAppFields{IsForceHTTPSEnabled: &stateBool}
	if !runtimeFieldsChanged(plan, state) {
		t.Error("expected true when IsForceHTTPSEnabled differs")
	}
}

func TestRuntimeFieldsChanged_Int64Change(t *testing.T) {
	t.Parallel()
	planInt := types.Int64Value(30)
	stateInt := types.Int64Value(5)
	plan := commonAppFields{HealthCheckInterval: &planInt}
	state := commonAppFields{HealthCheckInterval: &stateInt}
	if !runtimeFieldsChanged(plan, state) {
		t.Error("expected true when HealthCheckInterval differs")
	}
}

func TestRuntimeFieldsChanged_BothNil(t *testing.T) {
	t.Parallel()
	// All pointer fields are nil in both plan and state.
	plan := commonAppFields{}
	state := commonAppFields{}
	if runtimeFieldsChanged(plan, state) {
		t.Error("expected false when both plan and state have all nil fields")
	}
}

func TestRuntimeFieldsChanged_OneNilOneSet(t *testing.T) {
	t.Parallel()
	// One side nil, other side set: stringFieldChanged returns false for nil pairs.
	v := types.StringValue("x")
	plan := commonAppFields{Name: &v}
	state := commonAppFields{Name: nil}
	if runtimeFieldsChanged(plan, state) {
		t.Error("expected false when one side is nil (nil-safe guard)")
	}
}

func TestRuntimeFieldsChanged_NullValues(t *testing.T) {
	t.Parallel()
	null := types.StringNull()
	plan := commonAppFields{Domains: &null}
	state := commonAppFields{Domains: &null}
	if runtimeFieldsChanged(plan, state) {
		t.Error("expected false when both sides are null")
	}
}

// ---------------------------------------------------------------------------
// #359: buildUpdateInput
// ---------------------------------------------------------------------------

func TestBuildUpdateInput_Identical(t *testing.T) {
	t.Parallel()
	v := types.StringValue("app")
	d := types.StringValue("desc")
	dom := types.StringValue("https://app.example.com")
	mem := types.StringValue("0")
	memSwap := types.StringValue("0")
	memRes := types.StringValue("0")
	cpus := types.StringValue("0")
	cpuSet := types.StringValue("")
	hcPath := types.StringValue("/")
	hcPort := types.StringValue("")
	swap := types.Int64Value(60)
	shares := types.Int64Value(1024)
	hcInt := types.Int64Value(5)
	hcTO := types.Int64Value(5)
	hcRet := types.Int64Value(10)
	hcSP := types.Int64Value(5)
	hcRC := types.Int64Value(200)
	hcEnabled := types.BoolValue(false)
	autoD := types.BoolValue(true)
	hcCmd := types.StringValue("")
	hcHost := types.StringValue("localhost")
	hcMethod := types.StringValue("GET")
	hcResp := types.StringValue("")
	hcScheme := types.StringValue("http")
	hcType := types.StringValue("http")
	base := types.StringValue("")
	pub := types.StringValue("")
	drTag := types.StringValue("")
	dcd := types.StringValue("")
	gcs := types.StringValue("")
	wp := types.StringValue("")
	cdro := types.StringValue("")
	cl := types.StringValue("")
	cna := types.StringValue("")
	cnc := types.StringValue("")
	pm := types.StringValue("")
	red := types.StringValue("both")
	si := types.StringValue("nginx:alpine")
	authU := types.StringValue("")
	authP := types.StringValue("")
	pre := types.StringValue("")
	preC := types.StringValue("")
	post := types.StringValue("")
	postC := types.StringValue("")
	whBB := types.StringValue("")
	whGitea := types.StringValue("")
	whGH := types.StringValue("")
	whGL := types.StringValue("")
	isS := types.BoolValue(false)
	isSPA := types.BoolValue(false)
	isF := types.BoolValue(true)
	isAuth := types.BoolValue(false)
	conn := types.BoolValue(false)
	escape := types.BoolValue(true)
	pres := types.BoolValue(false)
	useBuild := types.BoolValue(false)
	fdo := types.BoolValue(false)

	f := commonAppFields{
		Name: &v, Description: &d, Domains: &dom,
		LimitsMemory: &mem, LimitsMemorySwap: &memSwap,
		LimitsMemoryReservation: &memRes, LimitsCPUs: &cpus,
		LimitsCPUSet: &cpuSet, LimitsMemorySwappiness: &swap,
		LimitsCPUShares:    &shares,
		HealthCheckEnabled: &hcEnabled, HealthCheckPath: &hcPath,
		HealthCheckPort: &hcPort, HealthCheckInterval: &hcInt,
		HealthCheckTimeout: &hcTO, HealthCheckRetries: &hcRet,
		HealthCheckStartPeriod: &hcSP, HealthCheckCommand: &hcCmd,
		HealthCheckHost: &hcHost, HealthCheckMethod: &hcMethod,
		HealthCheckResponseText: &hcResp, HealthCheckReturnCode: &hcRC,
		HealthCheckScheme: &hcScheme, HealthCheckType: &hcType,
		IsAutoDeployEnabled: &autoD,
		BaseDirectory:       &base, PublishDirectory: &pub,
		DockerRegistryImageTag: &drTag, DockerComposeDomains: &dcd,
		GitCommitSha: &gcs, WatchPaths: &wp,
		CustomDockerRunOptions: &cdro, CustomLabels: &cl,
		CustomNetworkAliases: &cna, CustomNginxConfiguration: &cnc,
		PortsMappings: &pm,
		Redirect:      &red, StaticImage: &si,
		IsStatic: &isS, IsSPA: &isSPA,
		IsForceHTTPSEnabled: &isF, IsHTTPBasicAuthEnabled: &isAuth,
		HTTPBasicAuthUsername: &authU, HTTPBasicAuthPassword: &authP,
		PreDeploymentCommand: &pre, PreDeploymentCommandContainer: &preC,
		PostDeploymentCommand: &post, PostDeploymentCommandContainer: &postC,
		ManualWebhookSecretBitbucket: &whBB, ManualWebhookSecretGitea: &whGitea,
		ManualWebhookSecretGitHub: &whGH, ManualWebhookSecretGitLab: &whGL,
		ConnectToDockerNetwork: &conn, IsContainerLabelEscapeEnabled: &escape,
		IsPreserveRepositoryEnabled: &pres, UseBuildServer: &useBuild,
		ForceDomainOverride: &fdo,
	}

	input := buildUpdateInput(f, f)

	// All diff fields should be nil when plan == state.
	if input.Name != nil {
		t.Errorf("expected Name nil, got %v", *input.Name)
	}
	if input.Description != nil {
		t.Errorf("expected Description nil, got %v", *input.Description)
	}
	if input.Domains != nil {
		t.Errorf("expected Domains nil, got %v", *input.Domains)
	}
	if input.LimitsMemory != nil {
		t.Errorf("expected LimitsMemory nil, got %v", *input.LimitsMemory)
	}
}

func TestBuildUpdateInput_SingleFieldChanged(t *testing.T) {
	t.Parallel()
	planName := types.StringValue("new-name")
	stateName := types.StringValue("old-name")
	desc := types.StringValue("same")
	dom := types.StringValue("")
	mem := types.StringValue("0")
	memSwap := types.StringValue("0")
	memRes := types.StringValue("0")
	cpus := types.StringValue("0")
	cpuSet := types.StringValue("")
	hcPath := types.StringValue("/")
	hcPort := types.StringValue("")
	swap := types.Int64Value(60)
	shares := types.Int64Value(1024)
	hcInt := types.Int64Value(5)
	hcTO := types.Int64Value(5)
	hcRet := types.Int64Value(10)
	hcSP := types.Int64Value(5)
	hcRC := types.Int64Value(200)
	hcEnabled := types.BoolValue(false)
	autoD := types.BoolValue(true)
	hcCmd := types.StringValue("")
	hcHost := types.StringValue("localhost")
	hcMethod := types.StringValue("GET")
	hcResp := types.StringValue("")
	hcScheme := types.StringValue("http")
	hcType := types.StringValue("http")
	base := types.StringValue("")
	pub := types.StringValue("")
	drTag := types.StringValue("")
	dcd := types.StringValue("")
	gcs := types.StringValue("")
	wp := types.StringValue("")
	cdro := types.StringValue("")
	cl := types.StringValue("")
	cna := types.StringValue("")
	cnc := types.StringValue("")
	pm := types.StringValue("")
	red := types.StringValue("both")
	si := types.StringValue("nginx:alpine")
	authU := types.StringValue("")
	authP := types.StringValue("")
	pre := types.StringValue("")
	preC := types.StringValue("")
	post := types.StringValue("")
	postC := types.StringValue("")
	whBB := types.StringValue("")
	whGitea := types.StringValue("")
	whGH := types.StringValue("")
	whGL := types.StringValue("")
	isS := types.BoolValue(false)
	isSPA := types.BoolValue(false)
	isF := types.BoolValue(true)
	isAuth := types.BoolValue(false)
	conn := types.BoolValue(false)
	escape := types.BoolValue(true)
	pres := types.BoolValue(false)
	useBuild := types.BoolValue(false)
	fdo := types.BoolValue(false)

	shared := commonAppFields{
		Description: &desc, Domains: &dom,
		LimitsMemory: &mem, LimitsMemorySwap: &memSwap,
		LimitsMemoryReservation: &memRes, LimitsCPUs: &cpus,
		LimitsCPUSet: &cpuSet, LimitsMemorySwappiness: &swap,
		LimitsCPUShares:    &shares,
		HealthCheckEnabled: &hcEnabled, HealthCheckPath: &hcPath,
		HealthCheckPort: &hcPort, HealthCheckInterval: &hcInt,
		HealthCheckTimeout: &hcTO, HealthCheckRetries: &hcRet,
		HealthCheckStartPeriod: &hcSP, HealthCheckCommand: &hcCmd,
		HealthCheckHost: &hcHost, HealthCheckMethod: &hcMethod,
		HealthCheckResponseText: &hcResp, HealthCheckReturnCode: &hcRC,
		HealthCheckScheme: &hcScheme, HealthCheckType: &hcType,
		IsAutoDeployEnabled: &autoD,
		BaseDirectory:       &base, PublishDirectory: &pub,
		DockerRegistryImageTag: &drTag, DockerComposeDomains: &dcd,
		GitCommitSha: &gcs, WatchPaths: &wp,
		CustomDockerRunOptions: &cdro, CustomLabels: &cl,
		CustomNetworkAliases: &cna, CustomNginxConfiguration: &cnc,
		PortsMappings: &pm,
		Redirect:      &red, StaticImage: &si,
		IsStatic: &isS, IsSPA: &isSPA,
		IsForceHTTPSEnabled: &isF, IsHTTPBasicAuthEnabled: &isAuth,
		HTTPBasicAuthUsername: &authU, HTTPBasicAuthPassword: &authP,
		PreDeploymentCommand: &pre, PreDeploymentCommandContainer: &preC,
		PostDeploymentCommand: &post, PostDeploymentCommandContainer: &postC,
		ManualWebhookSecretBitbucket: &whBB, ManualWebhookSecretGitea: &whGitea,
		ManualWebhookSecretGitHub: &whGH, ManualWebhookSecretGitLab: &whGL,
		ConnectToDockerNetwork: &conn, IsContainerLabelEscapeEnabled: &escape,
		IsPreserveRepositoryEnabled: &pres, UseBuildServer: &useBuild,
		ForceDomainOverride: &fdo,
	}

	plan := shared
	plan.Name = &planName
	state := shared
	state.Name = &stateName

	input := buildUpdateInput(plan, state)

	if input.Name == nil {
		t.Fatal("expected Name to be non-nil")
	}
	if *input.Name != "new-name" {
		t.Errorf("expected Name=%q, got %q", "new-name", *input.Name)
	}
	// Other fields should be nil.
	if input.Description != nil {
		t.Errorf("expected Description nil, got %v", *input.Description)
	}
}

func TestBuildUpdateInput_NilSafeOptionalPtrs(t *testing.T) {
	t.Parallel()
	// GitRepository nil in both sides should not panic and produce nil in output.
	name := types.StringValue("app")
	desc := types.StringValue("desc")
	dom := types.StringValue("")
	mem := types.StringValue("0")
	memSwap := types.StringValue("0")
	memRes := types.StringValue("0")
	cpus := types.StringValue("0")
	cpuSet := types.StringValue("")
	hcPath := types.StringValue("/")
	hcPort := types.StringValue("")
	swap := types.Int64Value(60)
	shares := types.Int64Value(1024)
	hcInt := types.Int64Value(5)
	hcTO := types.Int64Value(5)
	hcRet := types.Int64Value(10)
	hcSP := types.Int64Value(5)
	hcRC := types.Int64Value(200)
	hcEnabled := types.BoolValue(false)
	autoD := types.BoolValue(true)
	hcCmd := types.StringValue("")
	hcHost := types.StringValue("localhost")
	hcMethod := types.StringValue("GET")
	hcResp := types.StringValue("")
	hcScheme := types.StringValue("http")
	hcType := types.StringValue("http")
	base := types.StringValue("")
	pub := types.StringValue("")
	drTag := types.StringValue("")
	dcd := types.StringValue("")
	gcs := types.StringValue("")
	wp := types.StringValue("")
	cdro := types.StringValue("")
	cl := types.StringValue("")
	cna := types.StringValue("")
	cnc := types.StringValue("")
	pm := types.StringValue("")
	red := types.StringValue("both")
	si := types.StringValue("nginx:alpine")
	authU := types.StringValue("")
	authP := types.StringValue("")
	pre := types.StringValue("")
	preC := types.StringValue("")
	post := types.StringValue("")
	postC := types.StringValue("")
	whBB := types.StringValue("")
	whGitea := types.StringValue("")
	whGH := types.StringValue("")
	whGL := types.StringValue("")
	isS := types.BoolValue(false)
	isSPA := types.BoolValue(false)
	isF := types.BoolValue(true)
	isAuth := types.BoolValue(false)
	conn := types.BoolValue(false)
	escape := types.BoolValue(true)
	pres := types.BoolValue(false)
	useBuild := types.BoolValue(false)
	fdo := types.BoolValue(false)

	f := commonAppFields{
		Name: &name, Description: &desc, Domains: &dom,
		GitRepository: nil, GitBranch: nil, BuildPack: nil,
		PortsExposes: nil, InstallCommand: nil, BuildCommand: nil,
		StartCommand: nil, DockerfileLocation: nil,
		LimitsMemory: &mem, LimitsMemorySwap: &memSwap,
		LimitsMemoryReservation: &memRes, LimitsCPUs: &cpus,
		LimitsCPUSet: &cpuSet, LimitsMemorySwappiness: &swap,
		LimitsCPUShares:    &shares,
		HealthCheckEnabled: &hcEnabled, HealthCheckPath: &hcPath,
		HealthCheckPort: &hcPort, HealthCheckInterval: &hcInt,
		HealthCheckTimeout: &hcTO, HealthCheckRetries: &hcRet,
		HealthCheckStartPeriod: &hcSP, HealthCheckCommand: &hcCmd,
		HealthCheckHost: &hcHost, HealthCheckMethod: &hcMethod,
		HealthCheckResponseText: &hcResp, HealthCheckReturnCode: &hcRC,
		HealthCheckScheme: &hcScheme, HealthCheckType: &hcType,
		IsAutoDeployEnabled: &autoD,
		BaseDirectory:       &base, PublishDirectory: &pub,
		DockerRegistryImageTag: &drTag, DockerComposeDomains: &dcd,
		GitCommitSha: &gcs, WatchPaths: &wp,
		CustomDockerRunOptions: &cdro, CustomLabels: &cl,
		CustomNetworkAliases: &cna, CustomNginxConfiguration: &cnc,
		PortsMappings: &pm,
		Redirect:      &red, StaticImage: &si,
		IsStatic: &isS, IsSPA: &isSPA,
		IsForceHTTPSEnabled: &isF, IsHTTPBasicAuthEnabled: &isAuth,
		HTTPBasicAuthUsername: &authU, HTTPBasicAuthPassword: &authP,
		PreDeploymentCommand: &pre, PreDeploymentCommandContainer: &preC,
		PostDeploymentCommand: &post, PostDeploymentCommandContainer: &postC,
		ManualWebhookSecretBitbucket: &whBB, ManualWebhookSecretGitea: &whGitea,
		ManualWebhookSecretGitHub: &whGH, ManualWebhookSecretGitLab: &whGL,
		ConnectToDockerNetwork: &conn, IsContainerLabelEscapeEnabled: &escape,
		IsPreserveRepositoryEnabled: &pres, UseBuildServer: &useBuild,
		ForceDomainOverride: &fdo,
	}

	// Should not panic with nil optional pointers.
	input := buildUpdateInput(f, f)
	if input.GitRepository != nil {
		t.Error("expected GitRepository nil for nil optional ptrs")
	}
}

// ---------------------------------------------------------------------------
// #359: buildPostCreatePatch
// ---------------------------------------------------------------------------

func TestBuildPostCreatePatch_AllNil(t *testing.T) {
	t.Parallel()
	f := commonAppFields{}
	input := buildPostCreatePatch(f)
	if input.LimitsMemory != nil {
		t.Errorf("expected LimitsMemory nil, got %v", *input.LimitsMemory)
	}
	if input.HealthCheckEnabled != nil {
		t.Errorf("expected HealthCheckEnabled nil, got %v", *input.HealthCheckEnabled)
	}
}

func TestBuildPostCreatePatch_SetField(t *testing.T) {
	t.Parallel()
	mem := types.StringValue("512M")
	f := commonAppFields{LimitsMemory: &mem}
	input := buildPostCreatePatch(f)
	if input.LimitsMemory == nil {
		t.Fatal("expected LimitsMemory non-nil")
	}
	if *input.LimitsMemory != "512M" {
		t.Errorf("expected LimitsMemory=%q, got %q", "512M", *input.LimitsMemory)
	}
}

func TestBuildPostCreatePatch_NullField(t *testing.T) {
	t.Parallel()
	null := types.StringNull()
	f := commonAppFields{LimitsMemory: &null}
	input := buildPostCreatePatch(f)
	if input.LimitsMemory != nil {
		t.Errorf("expected LimitsMemory nil for null value, got %v", *input.LimitsMemory)
	}
}

func TestBuildPostCreatePatch_UnknownField(t *testing.T) {
	t.Parallel()
	unk := types.StringUnknown()
	f := commonAppFields{LimitsMemory: &unk}
	input := buildPostCreatePatch(f)
	if input.LimitsMemory != nil {
		t.Errorf("expected LimitsMemory nil for unknown value, got %v", *input.LimitsMemory)
	}
}

// ---------------------------------------------------------------------------
// #357: flattenApplicationCommon
// ---------------------------------------------------------------------------

// newDefaultFields returns a commonAppFields with all required pointer fields
// initialized to sensible defaults for flatten testing.
func newDefaultFields() (commonAppFields, *applicationCommonModel) {
	m := &applicationCommonModel{}
	f := m.common()
	return f, m
}

func TestFlattenApplicationCommon_BasicFields(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	app := &client.Application{
		UUID:            "uuid-1",
		Name:            "my-app",
		Description:     "test app",
		Domains:         "https://app.example.com",
		ProjectUUID:     "proj-1",
		ServerUUID:      "srv-1",
		EnvironmentName: "production",
		Status:          "running",
	}

	flattenApplicationCommon(app, f)

	if f.UUID.ValueString() != "uuid-1" {
		t.Errorf("UUID = %q, want %q", f.UUID.ValueString(), "uuid-1")
	}
	if f.Name.ValueString() != "my-app" {
		t.Errorf("Name = %q, want %q", f.Name.ValueString(), "my-app")
	}
	if f.Description.ValueString() != "test app" {
		t.Errorf("Description = %q, want %q", f.Description.ValueString(), "test app")
	}
	if f.ProjectUUID.ValueString() != "proj-1" {
		t.Errorf("ProjectUUID = %q, want %q", f.ProjectUUID.ValueString(), "proj-1")
	}
	if f.Status.ValueString() != "running" {
		t.Errorf("Status = %q, want %q", f.Status.ValueString(), "running")
	}
}

func TestFlattenApplicationCommon_NilAPIBoolDefaults(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	app := &client.Application{
		UUID:                "uuid-1",
		Name:                "app",
		HealthCheckEnabled:  nil, // API returns nil
		IsAutoDeployEnabled: nil, // API returns nil
	}

	flattenApplicationCommon(app, f)

	// HealthCheckEnabled nil -> defaults to false
	if f.HealthCheckEnabled.ValueBool() != false {
		t.Errorf("HealthCheckEnabled = %v, want false for nil API value", f.HealthCheckEnabled.ValueBool())
	}
}

func TestFlattenApplicationCommon_EmptyDockerfileLocationPreservesState(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	existing := types.StringValue("RnJvbSBub2RlOjIw")
	f.DockerfileLocation = &existing // Set the pointer (base model doesn't include it)

	app := &client.Application{
		UUID: "uuid-1",
		Name: "app",
		// DockerfileLocation is empty string from API (not returned on GET)
	}

	flattenApplicationCommon(app, f)

	// State should be preserved when API returns empty.
	if f.DockerfileLocation.ValueString() != "RnJvbSBub2RlOjIw" {
		t.Errorf("DockerfileLocation = %q, want preserved state value", f.DockerfileLocation.ValueString())
	}
}

func TestFlattenApplicationCommon_EmptyProjectUUIDPreservesState(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	existing := types.StringValue("proj-original")
	*f.ProjectUUID = existing

	app := &client.Application{
		UUID:        "uuid-1",
		Name:        "app",
		ProjectUUID: "", // API omits immutable field
	}

	flattenApplicationCommon(app, f)

	if f.ProjectUUID.ValueString() != "proj-original" {
		t.Errorf("ProjectUUID = %q, want preserved %q", f.ProjectUUID.ValueString(), "proj-original")
	}
}

func TestFlattenApplicationCommon_GitRepositoryNormalization(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	git := types.StringValue("https://github.com/org/repo")
	f.GitRepository = &git // Set the pointer (base model doesn't include it)

	app := &client.Application{
		UUID:          "uuid-1",
		Name:          "app",
		GitRepository: "org/repo", // Coolify strips the prefix
	}

	flattenApplicationCommon(app, f)

	// State has full URL, API returns bare slug; resolveGitRepository should preserve state.
	if f.GitRepository.ValueString() != "https://github.com/org/repo" {
		t.Errorf("GitRepository = %q, want preserved full URL", f.GitRepository.ValueString())
	}
}

func TestFlattenApplicationCommon_RedeployOnUpdatePreservation(t *testing.T) {
	t.Parallel()

	t.Run("null becomes false", func(t *testing.T) {
		t.Parallel()
		f, _ := newDefaultFields()
		null := types.BoolNull()
		*f.RedeployOnUpdate = null

		app := &client.Application{UUID: "uuid-1", Name: "app"}
		flattenApplicationCommon(app, f)

		if f.RedeployOnUpdate.ValueBool() != false {
			t.Errorf("RedeployOnUpdate = %v, want false for null", f.RedeployOnUpdate.ValueBool())
		}
	})

	t.Run("unknown becomes false", func(t *testing.T) {
		t.Parallel()
		f, _ := newDefaultFields()
		unk := types.BoolUnknown()
		*f.RedeployOnUpdate = unk

		app := &client.Application{UUID: "uuid-1", Name: "app"}
		flattenApplicationCommon(app, f)

		if f.RedeployOnUpdate.ValueBool() != false {
			t.Errorf("RedeployOnUpdate = %v, want false for unknown", f.RedeployOnUpdate.ValueBool())
		}
	})

	t.Run("true is preserved", func(t *testing.T) {
		t.Parallel()
		f, _ := newDefaultFields()
		tr := types.BoolValue(true)
		*f.RedeployOnUpdate = tr

		app := &client.Application{UUID: "uuid-1", Name: "app"}
		flattenApplicationCommon(app, f)

		if f.RedeployOnUpdate.ValueBool() != true {
			t.Errorf("RedeployOnUpdate = %v, want true (preserved)", f.RedeployOnUpdate.ValueBool())
		}
	})
}

// ---------------------------------------------------------------------------
// #357: flattenLimitsAndHealth
// ---------------------------------------------------------------------------

func TestFlattenLimitsAndHealth_NilHealthCheckEnabledDefault(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	app := &client.Application{HealthCheckEnabled: nil}

	flattenLimitsAndHealth(app, f)

	if f.HealthCheckEnabled.ValueBool() != false {
		t.Errorf("HealthCheckEnabled = %v, want false for nil API value", f.HealthCheckEnabled.ValueBool())
	}
}

func TestFlattenLimitsAndHealth_SetHealthCheckEnabled(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	tr := true
	app := &client.Application{HealthCheckEnabled: &tr}

	flattenLimitsAndHealth(app, f)

	if f.HealthCheckEnabled.ValueBool() != true {
		t.Errorf("HealthCheckEnabled = %v, want true", f.HealthCheckEnabled.ValueBool())
	}
}

func TestFlattenLimitsAndHealth_HealthCheckHostDefault(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	app := &client.Application{HealthCheckHost: ""}

	flattenLimitsAndHealth(app, f)

	if f.HealthCheckHost.ValueString() != "localhost" {
		t.Errorf("HealthCheckHost = %q, want %q", f.HealthCheckHost.ValueString(), "localhost")
	}
}

// ---------------------------------------------------------------------------
// #357: flattenExtendedDefaults
// ---------------------------------------------------------------------------

func TestFlattenExtendedDefaults_RedirectDefault(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	app := &client.Application{Redirect: ""}

	flattenExtendedDefaults(app, f)

	if f.Redirect.ValueString() != "both" {
		t.Errorf("Redirect = %q, want %q", f.Redirect.ValueString(), "both")
	}
}

func TestFlattenExtendedDefaults_InstantDeployNullDefaultsFalse(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	null := types.BoolNull()
	*f.InstantDeploy = null
	app := &client.Application{}

	flattenExtendedDefaults(app, f)

	if f.InstantDeploy.ValueBool() != false {
		t.Errorf("InstantDeploy = %v, want false for null", f.InstantDeploy.ValueBool())
	}
}

func TestFlattenExtendedDefaults_ConnectToDockerNetworkNilDefault(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	app := &client.Application{ConnectToDockerNetwork: nil}

	flattenExtendedDefaults(app, f)

	if f.ConnectToDockerNetwork.ValueBool() != false {
		t.Errorf("ConnectToDockerNetwork = %v, want false for nil API value", f.ConnectToDockerNetwork.ValueBool())
	}
}

func TestFlattenExtendedDefaults_BoolWithAPIValue(t *testing.T) {
	t.Parallel()
	f, _ := newDefaultFields()
	tr := true
	app := &client.Application{IsStatic: &tr}

	flattenExtendedDefaults(app, f)

	if f.IsStatic.ValueBool() != true {
		t.Errorf("IsStatic = %v, want true from API", f.IsStatic.ValueBool())
	}
}
