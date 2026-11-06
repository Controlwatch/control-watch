package surface

import "testing"

func TestVerifiedNamesAreUniqueWithinTheirKinds(t *testing.T) {
	t.Parallel()

	functions := []FunctionName{
		FunctionCompliance,
		FunctionIdentityVerifier,
		FunctionPaused,
		FunctionGetAdmin,
		FunctionGetExistingRoles,
		FunctionGetRoleMemberCount,
		FunctionGetRoleMember,
		FunctionGetModulesForHook,
		FunctionLinkedTokens,
		FunctionSACAdmin,
		FunctionVerifyIdentity,
		FunctionRecoveryTarget,
		FunctionTransferred,
		FunctionCreated,
		FunctionDestroyed,
		FunctionAllowance,
		FunctionApprove,
		FunctionBalance,
		FunctionBurn,
		FunctionBurnFrom,
		FunctionDecimals,
		FunctionTokenName,
		FunctionSymbol,
		FunctionTransfer,
		FunctionTransferFrom,
	}
	events := []EventName{
		EventComplianceSet,
		EventIdentityVerifierSet,
		EventModuleAdded,
		EventModuleRemoved,
		EventPaused,
		EventUnpaused,
		EventAdminTransferInitiated,
		EventAdminTransferCompleted,
		EventAdminRenounced,
		EventRoleAdminChanged,
		EventRoleGranted,
		EventRoleRevoked,
	}
	hooks := []ComplianceHook{HookTransferred, HookCreated, HookDestroyed}

	assertUnique(t, "function", functions)
	assertUnique(t, "event", events)
	assertUnique(t, "hook", hooks)
}

func assertUnique[T ~string](t *testing.T, kind string, values []T) {
	t.Helper()
	seen := make(map[T]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			t.Fatalf("%s name is empty", kind)
		}
		if _, exists := seen[value]; exists {
			t.Fatalf("duplicate %s name %q", kind, value)
		}
		seen[value] = struct{}{}
	}
}
