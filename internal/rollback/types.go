package rollback

import "time"

const (
	SchemaVersion = 1
	ToolName      = "kigrepair"

	WorkflowCleanup        = "cleanup"
	WorkflowRepair         = "repair"
	WorkflowInstall        = "install"
	WorkflowDefenderEnsure = "defender_ensure"

	TypeServiceStop          = "service_stop"
	TypeServiceDelete        = "service_delete"
	TypeProcessTerminate     = "process_terminate"
	TypeFileDelete           = "file_delete"
	TypeDirectoryDelete      = "directory_delete"
	TypeRegistryDelete       = "registry_delete"
	TypeDefenderAddExclusion = "defender_add_exclusion"
	TypeMSIUninstall         = "msi_uninstall"
	TypeMSIInstall           = "msi_install"
)

type SnapshotInput struct {
	ReportDir      string
	Workflow       string
	Detection      any
	InstallerPath  string
	HasInvite      bool
	IsAdmin        bool
	FileTargets    []string
	RegistryKeys   []string
	RequiredPaths  []string
	SnapshotOnly   bool
	PlannedChanges []PlannedChange
}

type RollbackInfo struct {
	SchemaVersion    int              `json:"schema_version"`
	Tool             string           `json:"tool"`
	Workflow         string           `json:"workflow"`
	CreatedAt        string           `json:"created_at"`
	ReportDir        string           `json:"report_dir"`
	SnapshotOnly     bool             `json:"snapshot_only"`
	RestoreSupported bool             `json:"restore_supported"`
	RestoreNotes     []string         `json:"restore_notes,omitempty"`
	HasInvite        bool             `json:"has_invite,omitempty"`
	InstallerPath    string           `json:"installer_path,omitempty"`
	IsAdmin          bool             `json:"is_admin"`
	Before           BeforeState      `json:"before"`
	PlannedChanges   []PlannedChange  `json:"planned_changes,omitempty"`
	ExecutedChanges  []ExecutedChange `json:"executed_changes,omitempty"`
	Warnings         []string         `json:"warnings,omitempty"`
	Errors           []string         `json:"errors,omitempty"`
}

type BeforeState struct {
	DetectionHealth string              `json:"detection_health,omitempty"`
	InstallMode     string              `json:"install_mode,omitempty"`
	InstallRoot     string              `json:"install_root,omitempty"`
	Services        []ServiceSnapshot   `json:"services,omitempty"`
	Processes       []ProcessSnapshot   `json:"processes,omitempty"`
	Files           []FileSnapshot      `json:"files,omitempty"`
	Directories     []DirectorySnapshot `json:"directories,omitempty"`
	RegistryKeys    []RegistrySnapshot  `json:"registry_keys,omitempty"`
	Defender        *DefenderSnapshot   `json:"defender,omitempty"`
	MSI             *MSISnapshot        `json:"msi,omitempty"`
}

type ServiceSnapshot struct {
	Name           string `json:"name"`
	Exists         bool   `json:"exists"`
	State          string `json:"state,omitempty"`
	StartType      string `json:"start_type,omitempty"`
	RawImagePath   string `json:"raw_image_path,omitempty"`
	ExecutablePath string `json:"executable_path,omitempty"`
	TrustLevel     string `json:"trust_level,omitempty"`
}

type ProcessSnapshot struct {
	PID            int    `json:"pid"`
	Name           string `json:"name"`
	ExecutablePath string `json:"executable_path,omitempty"`
	TrustLevel     string `json:"trust_level,omitempty"`
}

type FileSnapshot struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	SHA256    string `json:"sha256,omitempty"`
	Error     string `json:"error,omitempty"`
}

type DirectorySnapshot struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}

type RegistrySnapshot struct {
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	Accessible bool   `json:"accessible"`
	Error      string `json:"error,omitempty"`
}

type DefenderSnapshot struct {
	Status               string   `json:"status,omitempty"`
	Exclusions           []string `json:"exclusions,omitempty"`
	NormalizedExclusions []string `json:"normalized_exclusions,omitempty"`
	RequiredPath         string   `json:"required_path,omitempty"`
	Covered              bool     `json:"covered"`
	CoveredBy            string   `json:"covered_by,omitempty"`
}

type MSISnapshot struct {
	ProductCode  string `json:"product_code,omitempty"`
	PackedCode   string `json:"packed_code,omitempty"`
	Installed    bool   `json:"installed"`
	UninstallKey string `json:"uninstall_key,omitempty"`
}

type PlannedChange struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Target      string   `json:"target"`
	Destructive bool     `json:"destructive"`
	Reason      string   `json:"reason,omitempty"`
	Source      string   `json:"source,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

type ExecutedChange struct {
	PlannedID  string   `json:"planned_id,omitempty"`
	Type       string   `json:"type"`
	Target     string   `json:"target"`
	Status     string   `json:"status"`
	StartedAt  string   `json:"started_at,omitempty"`
	FinishedAt string   `json:"finished_at,omitempty"`
	Message    string   `json:"message,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

type ChangeLedger struct {
	info *RollbackInfo
}

type Summary struct {
	SnapshotCreated  bool     `json:"snapshot_created"`
	Path             string   `json:"path,omitempty"`
	RestoreSupported bool     `json:"restore_supported"`
	PlannedChanges   int      `json:"planned_changes"`
	ExecutedChanges  int      `json:"executed_changes"`
	Warnings         []string `json:"warnings,omitempty"`
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339)
}
