package app

type WorkflowCatalog struct {
	Workflows []WorkflowCatalogEntry `json:"workflows"`
}

type WorkflowCatalogEntry struct {
	Name               string   `json:"name"`
	DisplayName        string   `json:"display_name"`
	Description        string   `json:"description"`
	ReadOnly           bool     `json:"read_only"`
	Mutating           bool     `json:"mutating"`
	RequiresAdmin      bool     `json:"requires_admin"`
	RequiresInstaller  bool     `json:"requires_installer"`
	RequiresInvite     bool     `json:"requires_invite"`
	SupportsDryRun     bool     `json:"supports_dry_run"`
	SupportsProfile    bool     `json:"supports_profile"`
	PrimaryResultFiles []string `json:"primary_result_files"`
	SafetyLevel        string   `json:"safety_level"`
}

func DefaultWorkflowCatalog() WorkflowCatalog {
	return WorkflowCatalog{Workflows: []WorkflowCatalogEntry{
		{Name: "check", DisplayName: "Check", Description: "Detect Grabber installation state.", ReadOnly: true, SupportsProfile: true, PrimaryResultFiles: []string{"initial-detection.json", "classification-result.json", "recommendation-result.json"}, SafetyLevel: "read_only"},
		{Name: "verify", DisplayName: "Verify", Description: "Verify current Grabber installation health.", ReadOnly: true, SupportsProfile: true, PrimaryResultFiles: []string{"verification-result.json"}, SafetyLevel: "read_only"},
		{Name: "preflight", DisplayName: "Preflight", Description: "Check repair readiness without modifying the endpoint.", ReadOnly: true, RequiresInstaller: true, RequiresInvite: true, SupportsProfile: true, PrimaryResultFiles: []string{"preflight-result.json"}, SafetyLevel: "read_only"},
		{Name: "repair_dry_run", DisplayName: "Repair Dry-Run", Description: "Build a repair plan without changing the endpoint.", ReadOnly: true, RequiresInstaller: true, RequiresInvite: true, SupportsDryRun: true, SupportsProfile: true, PrimaryResultFiles: []string{"repair-plan.json"}, SafetyLevel: "read_only"},
		{Name: "repair", DisplayName: "Repair", Description: "Execute validated cleanup, installation, Defender ensure, and verification.", Mutating: true, RequiresAdmin: true, RequiresInstaller: true, RequiresInvite: true, SupportsDryRun: true, SupportsProfile: true, PrimaryResultFiles: []string{"repair-result.json", "rollback-info.json"}, SafetyLevel: "destructive"},
		{Name: "cleanup_dry_run", DisplayName: "Cleanup Dry-Run", Description: "Build a cleanup plan without deleting files, services, or registry keys.", ReadOnly: true, SupportsDryRun: true, SupportsProfile: true, PrimaryResultFiles: []string{"cleanup-plan.json"}, SafetyLevel: "read_only"},
		{Name: "cleanup", DisplayName: "Cleanup", Description: "Execute a validated cleanup plan.", Mutating: true, RequiresAdmin: true, SupportsDryRun: true, SupportsProfile: true, PrimaryResultFiles: []string{"cleanup-result.json", "rollback-info.json"}, SafetyLevel: "destructive"},
		{Name: "collect_report", DisplayName: "Collect Report", Description: "Collect diagnostics and create a support bundle.", ReadOnly: true, SupportsProfile: true, PrimaryResultFiles: []string{"collect-result.json", "kigrepair-support-bundle.zip"}, SafetyLevel: "read_only"},
		{Name: "reports_list", DisplayName: "Reports List", Description: "List kigrepair report directories.", ReadOnly: true, SupportsProfile: true, PrimaryResultFiles: []string{"reports-list.json"}, SafetyLevel: "read_only"},
		{Name: "reports_cleanup_dry_run", DisplayName: "Reports Cleanup Dry-Run", Description: "Plan report directory retention cleanup.", ReadOnly: true, SupportsDryRun: true, SupportsProfile: true, PrimaryResultFiles: []string{"report-cleanup-plan.json"}, SafetyLevel: "read_only"},
		{Name: "reports_cleanup", DisplayName: "Reports Cleanup", Description: "Delete validated old kigrepair report directories.", Mutating: true, SupportsDryRun: true, SupportsProfile: true, PrimaryResultFiles: []string{"report-cleanup-result.json"}, SafetyLevel: "destructive"},
		{Name: "config_show", DisplayName: "Config Show", Description: "Show effective configuration.", ReadOnly: true, SupportsProfile: true, PrimaryResultFiles: []string{"config-show.json"}, SafetyLevel: "read_only"},
		{Name: "config_validate", DisplayName: "Config Validate", Description: "Validate effective configuration.", ReadOnly: true, SupportsProfile: true, PrimaryResultFiles: []string{"config-validation.json"}, SafetyLevel: "read_only"},
	}}
}
