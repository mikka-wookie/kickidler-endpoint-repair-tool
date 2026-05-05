package reports

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"kigrepair/internal/safety"
	"kigrepair/internal/version"
)

const SupportBundleName = "kigrepair-support-bundle.zip"

const bundleRoot = "kigrepair-support-bundle"

type ArchiveResult struct {
	Path     string       `json:"path"`
	Files    []string     `json:"files"`
	Included []BundleFile `json:"included,omitempty"`
	Skipped  []string     `json:"skipped,omitempty"`
	Warnings []string     `json:"warnings,omitempty"`
	Errors   []string     `json:"errors,omitempty"`
}

type BundleManifest struct {
	Tool             string       `json:"tool"`
	Build            version.Info `json:"build"`
	BundleVersion    int          `json:"bundle_version"`
	CreatedAt        time.Time    `json:"created_at"`
	ReportDir        string       `json:"report_dir"`
	RedactionEnabled bool         `json:"redaction_enabled"`
	Files            []BundleFile `json:"files"`
	Warnings         []string     `json:"warnings"`
	Errors           []string     `json:"errors"`
}

type BundleFile struct {
	Path      string `json:"path"`
	Source    string `json:"source"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

type bundleEntry struct {
	manifestPath string
	source       string
	content      []byte
}

func CreateSupportBundle(dir string) (ArchiveResult, error) {
	result, entries, manifest := buildSupportBundle(dir)
	file, err := os.Create(result.Path)
	if err != nil {
		return result, err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	if err := addManifest(writer, manifest); err != nil {
		_ = writer.Close()
		return result, err
	}
	for _, entry := range entries {
		if err := addZipContent(writer, entry.manifestPath, entry.content); err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}
	}
	if err := writer.Close(); err != nil {
		return result, err
	}
	return result, nil
}

func InspectSupportBundle(dir string) ArchiveResult {
	result, _, _ := buildSupportBundle(dir)
	return result
}

func buildSupportBundle(dir string) (ArchiveResult, []bundleEntry, BundleManifest) {
	result := ArchiveResult{Path: filepath.Join(dir, SupportBundleName)}
	entries, skipped, warnings, errorsList := collectBundleEntries(dir, result.Path)
	result.Skipped = append(result.Skipped, skipped...)
	result.Warnings = append(result.Warnings, warnings...)
	result.Errors = append(result.Errors, errorsList...)

	manifest := BundleManifest{
		Tool:             version.ToolName,
		Build:            version.Get(),
		BundleVersion:    1,
		CreatedAt:        time.Now().UTC(),
		ReportDir:        dir,
		RedactionEnabled: true,
		Warnings:         append([]string{}, result.Warnings...),
		Errors:           append([]string{}, result.Errors...),
	}

	for _, entry := range entries {
		sum := sha256.Sum256(entry.content)
		file := BundleFile{
			Path:      entry.manifestPath,
			Source:    entry.source,
			SizeBytes: int64(len(entry.content)),
			SHA256:    hex.EncodeToString(sum[:]),
		}
		manifest.Files = append(manifest.Files, file)
		result.Files = append(result.Files, entry.manifestPath)
		result.Included = append(result.Included, file)
	}

	sort.Slice(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Path < manifest.Files[j].Path
	})
	result.Included = append([]BundleFile(nil), manifest.Files...)
	sort.Strings(result.Files)
	return result, entries, manifest
}

func collectBundleEntries(reportDir string, bundlePath string) ([]bundleEntry, []string, []string, []string) {
	specs := bundleSpecs()
	entries := make([]bundleEntry, 0, len(specs))
	skipped := make([]string, 0)
	warnings := make([]string, 0)
	errorsList := make([]string, 0)
	for _, spec := range specs {
		source := filepath.Join(reportDir, filepath.FromSlash(spec.sourceRel))
		if samePath(source, bundlePath) {
			continue
		}
		data, err := os.ReadFile(source)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				warnings = append(warnings, "optional file missing: "+spec.sourceRel)
				skipped = append(skipped, spec.archiveRel)
				continue
			}
			warnings = append(warnings, fmt.Sprintf("optional file skipped: %s: %v", spec.sourceRel, err))
			skipped = append(skipped, spec.archiveRel)
			continue
		}
		if !isTextBundleFile(source) {
			warnings = append(warnings, "non-text file skipped: "+spec.sourceRel)
			skipped = append(skipped, spec.archiveRel)
			continue
		}
		archivePath, err := cleanArchivePath(spec.archiveRel)
		if err != nil {
			errorsList = append(errorsList, fmt.Sprintf("invalid archive path %q: %v", spec.archiveRel, err))
			continue
		}
		entries = append(entries, bundleEntry{
			manifestPath: archivePath,
			source:       source,
			content:      safety.RedactBytes(data),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].manifestPath < entries[j].manifestPath
	})
	sort.Strings(skipped)
	return entries, skipped, warnings, errorsList
}

type bundleSpec struct {
	sourceRel  string
	archiveRel string
}

func bundleSpecs() []bundleSpec {
	reportNames := []string{
		"wizard-result.json",
		"repair-plan.json",
		"preflight-result.json",
		"initial-detection.json",
		"final-detection.json",
		"operations.json",
		"rollback-info.json",
		"cleanup-plan.json",
		"cleanup-result.json",
		"install-result.json",
		"defender-result.json",
		"repair-result.json",
		"verification-result.json",
		"preflight-result.json",
		"classification-result.json",
		"recommendation-result.json",
		"classification-result.json",
		"installer-validation.json",
		"collect-result.json",
	}
	specs := []bundleSpec{{sourceRel: "summary.txt", archiveRel: "summary.txt"}}
	for _, name := range reportNames {
		specs = append(specs, bundleSpec{sourceRel: name, archiveRel: "reports/" + name})
	}
	for _, name := range []string{"repair.log", "msi-install.log", "msi-uninstall.log"} {
		specs = append(specs, bundleSpec{sourceRel: name, archiveRel: "logs/" + name})
	}
	for _, name := range []string{"config.json", "environment.json", "services.json", "processes.json", "defender.json", "registry.json"} {
		specs = append(specs, bundleSpec{sourceRel: "system/" + name, archiveRel: "system/" + name})
	}
	return specs
}

func addManifest(writer *zip.Writer, manifest BundleManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return addZipContent(writer, "manifest.json", data)
}

func addZipContent(writer *zip.Writer, archivePath string, content []byte) error {
	archivePath, err := cleanArchivePath(archivePath)
	if err != nil {
		return err
	}
	header := &zip.FileHeader{
		Name:   bundleRoot + "/" + archivePath,
		Method: zip.Deflate,
	}
	header.SetMode(0644)
	header.Modified = time.Unix(0, 0).UTC()
	target, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = target.Write(content)
	return err
}

func cleanArchivePath(path string) (string, error) {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return "", errors.New("empty archive path")
	}
	if strings.HasPrefix(path, "/") || filepath.IsAbs(path) {
		return "", errors.New("absolute archive path rejected")
	}
	if len(path) >= 2 && path[1] == ':' {
		return "", errors.New("absolute archive path rejected")
	}
	if path == ".." || strings.HasPrefix(path, "../") || strings.Contains(path, "/../") {
		return "", errors.New("parent traversal rejected")
	}
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." || cleaned == "" {
		return "", errors.New("empty archive path")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", errors.New("parent traversal rejected")
	}
	return cleaned, nil
}

func isTextBundleFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".log", ".json", ".xml", ".csv", ".ps1", ".bat", ".cmd":
		return true
	default:
		return false
	}
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
