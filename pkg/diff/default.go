package diff

// NewDefaultRunner constructs a runner containing every ControlWatch detector.
func NewDefaultRunner() *Runner {
	return NewRunner(
		ComplianceContractChanged{},
		IdentityVerifierChanged{},
		ModuleAdded{},
		ModuleRemoved{},
		WasmHashChanged{},
		AdminChanged{},
		RoleMembersChanged{},
		PauseStateChanged{},
		SACAdminChanged{},
		ClassificationChanged{},
		ClassicFlagsChanged{},
		ModuleHooksChanged{},
	)
}
