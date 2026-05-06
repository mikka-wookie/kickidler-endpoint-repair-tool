package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type RunMetadata struct {
	RunID           string    `json:"run_id"`
	WorkflowID      string    `json:"workflow_id,omitempty"`
	WorkflowName    string    `json:"workflow"`
	CommandName     string    `json:"command"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at,omitempty"`
	DurationMS      int64     `json:"duration_ms,omitempty"`
	ReportDir       string    `json:"report_dir,omitempty"`
	Version         string    `json:"version,omitempty"`
	Commit          string    `json:"commit,omitempty"`
	BuildDate       string    `json:"build_date,omitempty"`
	UserInteractive bool      `json:"user_interactive"`
	DryRun          bool      `json:"dry_run"`
	ReadOnly        bool      `json:"read_only"`
	Elevated        bool      `json:"elevated,omitempty"`
	CorrelationID   string    `json:"correlation_id"`
}

func NewRunID(now time.Time) string {
	var random [3]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Sprintf("kigrun-%s-%d", now.Format("20060102-150405"), now.UnixNano()%1000000)
	}
	return "kigrun-" + now.Format("20060102-150405") + "-" + hex.EncodeToString(random[:])
}

func Slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "operation"
	}
	return slug
}
