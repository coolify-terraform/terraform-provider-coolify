package scheduledtask

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplyScheduledTask_DropsUnsentFalseOnOldCoolify(t *testing.T) {
	t.Parallel()
	state := &scheduledTaskResourceModel{UseShell: types.BoolValue(false)}
	applyScheduledTask(client.ScheduledTask{
		Name: "n", Command: "echo hi", Frequency: "0 * * * *", Enabled: true,
	}, state, false)
	if !state.UseShell.IsNull() {
		t.Fatalf("use_shell = %#v, want null", state.UseShell)
	}
}

func TestApplyScheduledTask_KeepsUnsentTrueOnOldCoolify(t *testing.T) {
	t.Parallel()
	state := &scheduledTaskResourceModel{UseShell: types.BoolValue(true)}
	applyScheduledTask(client.ScheduledTask{
		Name: "n", Command: "echo hi", Frequency: "0 * * * *", Enabled: true,
	}, state, false)
	if state.UseShell.IsNull() || !state.UseShell.ValueBool() {
		t.Fatalf("use_shell = %#v, want true", state.UseShell)
	}
}

func TestApplyScheduledTask_KeepsFalseWhenServerAccepts(t *testing.T) {
	t.Parallel()
	state := &scheduledTaskResourceModel{UseShell: types.BoolValue(false)}
	applyScheduledTask(client.ScheduledTask{
		Name: "n", Command: "echo hi", Frequency: "0 * * * *", Enabled: true,
	}, state, true)
	if state.UseShell.IsNull() || state.UseShell.ValueBool() {
		t.Fatalf("use_shell = %#v, want false", state.UseShell)
	}
}
