// Package surface contains every contract-specific name understood by
// ControlWatch. Other packages must refer to these constants instead of
// embedding ABI function names, event topics, or enum variants.
package surface

// FunctionName is an exported Soroban contract function name.
type FunctionName string

// EventName is the symbol in topic segment zero of a contract event.
type EventName string

// ComplianceHook is an OpenZeppelin RWA compliance-hook variant.
type ComplianceHook string

const (
	// Verified RWA and access-control getters.
	FunctionCompliance         FunctionName = "compliance"
	FunctionIdentityVerifier   FunctionName = "identity_verifier"
	FunctionPaused             FunctionName = "paused"
	FunctionGetAdmin           FunctionName = "get_admin"
	FunctionGetExistingRoles   FunctionName = "get_existing_roles"
	FunctionGetRoleMemberCount FunctionName = "get_role_member_count"
	FunctionGetRoleMember      FunctionName = "get_role_member"

	// Verified compliance-contract getters.
	FunctionGetModulesForHook FunctionName = "get_modules_for_hook"
	FunctionLinkedTokens      FunctionName = "linked_tokens"

	// Verified Stellar Asset Contract getter.
	FunctionSACAdmin FunctionName = "admin"

	// RWA collaboration functions used for surface classification.
	FunctionVerifyIdentity FunctionName = "verify_identity"
	FunctionRecoveryTarget FunctionName = "recovery_target"
	FunctionTransferred    FunctionName = "transferred"
	FunctionCreated        FunctionName = "created"
	FunctionDestroyed      FunctionName = "destroyed"

	// SEP-41 functions used for surface classification.
	FunctionAllowance    FunctionName = "allowance"
	FunctionApprove      FunctionName = "approve"
	FunctionBalance      FunctionName = "balance"
	FunctionBurn         FunctionName = "burn"
	FunctionBurnFrom     FunctionName = "burn_from"
	FunctionDecimals     FunctionName = "decimals"
	FunctionTokenName    FunctionName = "name"
	FunctionSymbol       FunctionName = "symbol"
	FunctionTransfer     FunctionName = "transfer"
	FunctionTransferFrom FunctionName = "transfer_from"
)

const (
	EventComplianceSet          EventName = "compliance_set"
	EventIdentityVerifierSet    EventName = "identity_verifier_set"
	EventModuleAdded            EventName = "module_added"
	EventModuleRemoved          EventName = "module_removed"
	EventPaused                 EventName = "paused"
	EventUnpaused               EventName = "unpaused"
	EventAdminTransferInitiated EventName = "admin_transfer_initiated"
	EventAdminTransferCompleted EventName = "admin_transfer_completed"
	EventAdminRenounced         EventName = "admin_renounced"
	EventRoleAdminChanged       EventName = "role_admin_changed"
	EventRoleGranted            EventName = "role_granted"
	EventRoleRevoked            EventName = "role_revoked"
)

const (
	HookTransferred ComplianceHook = "Transferred"
	HookCreated     ComplianceHook = "Created"
	HookDestroyed   ComplianceHook = "Destroyed"
)
