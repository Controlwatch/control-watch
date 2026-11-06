package diff

import (
	"reflect"
	"testing"
)

func TestNewDefaultRunnerIncludesEveryDetectorInCodeOrder(t *testing.T) {
	t.Parallel()

	runner := NewDefaultRunner()
	want := []reflect.Type{
		reflect.TypeOf(ComplianceContractChanged{}),
		reflect.TypeOf(IdentityVerifierChanged{}),
		reflect.TypeOf(ModuleAdded{}),
		reflect.TypeOf(ModuleRemoved{}),
		reflect.TypeOf(WasmHashChanged{}),
		reflect.TypeOf(AdminChanged{}),
		reflect.TypeOf(RoleMembersChanged{}),
		reflect.TypeOf(PauseStateChanged{}),
		reflect.TypeOf(SACAdminChanged{}),
		reflect.TypeOf(ClassificationChanged{}),
		reflect.TypeOf(ClassicFlagsChanged{}),
		reflect.TypeOf(ModuleHooksChanged{}),
	}
	if len(runner.detectors) != len(want) {
		t.Fatalf("detector count = %d, want %d", len(runner.detectors), len(want))
	}
	for index, detector := range runner.detectors {
		if got := reflect.TypeOf(detector); got != want[index] {
			t.Fatalf("detector %d = %v, want %v", index, got, want[index])
		}
	}
}
