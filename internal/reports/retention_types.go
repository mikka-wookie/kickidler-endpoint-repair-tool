package reports

type ReportEntry struct {
	Name             string   `json:"name"`
	Path             string   `json:"path"`
	CreatedAt        string   `json:"created_at,omitempty"`
	ModifiedAt       string   `json:"modified_at,omitempty"`
	SizeBytes        int64    `json:"size_bytes"`
	HasSupportBundle bool     `json:"has_support_bundle"`
	Workflow         string   `json:"workflow,omitempty"`
	Status           string   `json:"status,omitempty"`
	Files            []string `json:"files,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
}

type ReportListResult struct {
	Command        string        `json:"command"`
	ReportsRoot    string        `json:"reports_root"`
	Count          int           `json:"count"`
	TotalSizeBytes int64         `json:"total_size_bytes"`
	Reports        []ReportEntry `json:"reports"`
	Warnings       []string      `json:"warnings,omitempty"`
	Errors         []string      `json:"errors,omitempty"`
}

type ReportCleanupPlan struct {
	SchemaVersion    int                  `json:"schema_version"`
	CreatedAt        string               `json:"created_at"`
	ReportsRoot      string               `json:"reports_root"`
	ReportDir        string               `json:"report_dir,omitempty"`
	DryRun           bool                 `json:"dry_run"`
	OlderThan        string               `json:"older_than"`
	KeepLast         int                  `json:"keep_last"`
	Candidates       []ReportEntry        `json:"candidates"`
	PlannedDeletes   []ReportDeleteTarget `json:"planned_deletes"`
	Skipped          []ReportSkip         `json:"skipped,omitempty"`
	TotalSizeBytes   int64                `json:"total_size_bytes"`
	PlannedFreeBytes int64                `json:"planned_free_bytes"`
	Warnings         []string             `json:"warnings,omitempty"`
	Errors           []string             `json:"errors,omitempty"`
}

type ReportDeleteTarget struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	SizeBytes  int64  `json:"size_bytes"`
	ModifiedAt string `json:"modified_at,omitempty"`
	Reason     string `json:"reason"`
	Validated  bool   `json:"validated"`
}

type ReportSkip struct {
	Name   string `json:"name,omitempty"`
	Path   string `json:"path,omitempty"`
	Reason string `json:"reason"`
}

type ReportCleanupResult struct {
	Command          string                `json:"command"`
	Status           string                `json:"status"`
	ExitCode         int                   `json:"exit_code"`
	ReportsRoot      string                `json:"reports_root"`
	ReportDir        string                `json:"report_dir,omitempty"`
	DryRun           bool                  `json:"dry_run"`
	PlanPath         string                `json:"plan_path,omitempty"`
	PlannedDeletes   int                   `json:"planned_deletes"`
	DeletedCount     int                   `json:"deleted"`
	Deleted          []ReportDeleteOutcome `json:"deleted,omitempty"`
	Skipped          []ReportSkip          `json:"skipped,omitempty"`
	PlannedFreeBytes int64                 `json:"planned_free_bytes"`
	FreedBytes       int64                 `json:"freed_bytes"`
	Warnings         []string              `json:"warnings,omitempty"`
	Errors           []string              `json:"errors,omitempty"`
}

type ReportDeleteOutcome struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Status    string   `json:"status"`
	SizeBytes int64    `json:"size_bytes,omitempty"`
	Message   string   `json:"message,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

type ReportListOptions struct {
	Limit int
	All   bool
}

type ReportCleanupOptions struct {
	OlderThan       string
	KeepLast        int
	DryRun          bool
	Yes             bool
	ActiveReportDir string
	Now             string
}
