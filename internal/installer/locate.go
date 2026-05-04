package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"kigrepair/internal/detector"
)

type InstallerSource string

const (
	InstallerSourceExplicit   InstallerSource = "explicit"
	InstallerSourceExeDir     InstallerSource = "exe_dir"
	InstallerSourceWorkDir    InstallerSource = "work_dir"
	InstallerSourceWorkAssets InstallerSource = "work_assets"
	InstallerSourceExeAssets  InstallerSource = "exe_assets"
	InstallerSourceNotFound   InstallerSource = "not_found"
)

type InstallerCandidate struct {
	Path          string          `json:"path"`
	Name          string          `json:"name"`
	Source        InstallerSource `json:"source"`
	Exists        bool            `json:"exists"`
	PreferredRank int             `json:"preferred_rank"`
	Architecture  string          `json:"architecture"`
	PackageType   string          `json:"package_type"`
}

type InstallerResolution struct {
	ExplicitPath   string                `json:"explicit_path,omitempty"`
	SelectedPath   string                `json:"selected_path,omitempty"`
	SelectedSource InstallerSource       `json:"selected_source"`
	Candidates     []InstallerCandidate  `json:"candidates"`
	SearchPaths    []InstallerSearchPath `json:"search_paths,omitempty"`
	OSArchitecture string                `json:"os_architecture"`
	Error          string                `json:"error,omitempty"`
}

type InstallerSearchPath struct {
	Dir    string          `json:"dir"`
	Source InstallerSource `json:"source"`
}

var ErrInstallerNotFound = errors.New("no supported Grabber MSI installer found")

func ResolveInstaller(explicitPath string) (InstallerResolution, error) {
	exePath, exeErr := os.Executable()
	workDir, workErr := os.Getwd()

	search := []InstallerSearchPath{}
	if exeErr == nil {
		exeDir := filepath.Dir(exePath)
		search = append(search, InstallerSearchPath{Dir: exeDir, Source: InstallerSourceExeDir})
	}
	if workErr == nil {
		search = append(search, InstallerSearchPath{Dir: workDir, Source: InstallerSourceWorkDir})
		search = append(search, InstallerSearchPath{Dir: filepath.Join(workDir, "assets"), Source: InstallerSourceWorkAssets})
	}
	if exeErr == nil {
		search = append(search, InstallerSearchPath{Dir: filepath.Join(filepath.Dir(exePath), "assets"), Source: InstallerSourceExeAssets})
	}

	return ResolveInstallerWithSearchPaths(explicitPath, search, runtime.GOARCH)
}

func ResolveInstallerWithSearchPaths(explicitPath string, searchPaths []InstallerSearchPath, osArch string) (InstallerResolution, error) {
	resolution := InstallerResolution{
		ExplicitPath:   strings.TrimSpace(explicitPath),
		SelectedSource: InstallerSourceNotFound,
		SearchPaths:    append([]InstallerSearchPath(nil), searchPaths...),
		OSArchitecture: normalizeOSArchitecture(osArch),
	}

	if strings.TrimSpace(explicitPath) != "" {
		path, err := ValidateInstallerPath(explicitPath)
		if err != nil {
			resolution.Error = err.Error()
			return resolution, err
		}
		name := filepath.Base(path)
		packageType, arch, ok := ParseInstallerName(name)
		if !ok {
			packageType = "custom"
			arch = "unknown"
		}
		resolution.SelectedPath = path
		resolution.SelectedSource = InstallerSourceExplicit
		resolution.Candidates = []InstallerCandidate{{
			Path:          path,
			Name:          name,
			Source:        InstallerSourceExplicit,
			Exists:        true,
			PreferredRank: preferredRank(name, PreferredInstallerNames(resolution.OSArchitecture)),
			Architecture:  arch,
			PackageType:   packageType,
		}}
		return resolution, nil
	}

	preferredNames := PreferredInstallerNames(resolution.OSArchitecture)
	for _, searchPath := range searchPaths {
		dir := strings.TrimSpace(searchPath.Dir)
		if dir == "" {
			continue
		}
		for _, name := range preferredNames {
			path := filepath.Join(dir, name)
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			packageType, arch, ok := ParseInstallerName(name)
			if !ok {
				continue
			}
			abs, err := filepath.Abs(path)
			if err == nil {
				path = detector.NormalizeWindowsPath(abs)
			}
			resolution.Candidates = append(resolution.Candidates, InstallerCandidate{
				Path:          path,
				Name:          name,
				Source:        searchPath.Source,
				Exists:        true,
				PreferredRank: preferredRank(name, preferredNames),
				Architecture:  arch,
				PackageType:   packageType,
			})
		}
	}

	sort.SliceStable(resolution.Candidates, func(i, j int) bool {
		left := resolution.Candidates[i]
		right := resolution.Candidates[j]
		if left.PreferredRank != right.PreferredRank {
			return left.PreferredRank < right.PreferredRank
		}
		return sourceRank(left.Source) < sourceRank(right.Source)
	})

	if len(resolution.Candidates) == 0 {
		err := fmt.Errorf("%w in supported search paths", ErrInstallerNotFound)
		resolution.Error = err.Error()
		return resolution, err
	}

	selected := resolution.Candidates[0]
	resolution.SelectedPath = selected.Path
	resolution.SelectedSource = selected.Source
	return resolution, nil
}

func PreferredInstallerNames(osArch string) []string {
	switch normalizeOSArchitecture(osArch) {
	case "386", "x86", "x32":
		return []string{
			"grabberEM.x32.msi",
			"grabberTT.x32.msi",
			"grabberEM.x64.msi",
			"grabberTT.x64.msi",
			"grabber.msi",
		}
	default:
		return []string{
			"grabberEM.x64.msi",
			"grabberTT.x64.msi",
			"grabberEM.x32.msi",
			"grabberTT.x32.msi",
			"grabber.msi",
		}
	}
}

func ParseInstallerName(name string) (packageType string, arch string, ok bool) {
	switch strings.ToLower(filepath.Base(name)) {
	case "grabberem.x64.msi":
		return "EM", "x64", true
	case "grabberem.x32.msi":
		return "EM", "x32", true
	case "grabbertt.x64.msi":
		return "TT", "x64", true
	case "grabbertt.x32.msi":
		return "TT", "x32", true
	case "grabber.msi":
		return "legacy", "unknown", true
	default:
		return "", "", false
	}
}

func normalizeOSArchitecture(osArch string) string {
	arch := strings.ToLower(strings.TrimSpace(osArch))
	switch arch {
	case "amd64", "x64":
		return "amd64"
	case "386", "x86", "x32":
		return "386"
	default:
		return arch
	}
}

func preferredRank(name string, preferredNames []string) int {
	for index, preferred := range preferredNames {
		if strings.EqualFold(name, preferred) {
			return index + 1
		}
	}
	return len(preferredNames) + 1
}

func sourceRank(source InstallerSource) int {
	switch source {
	case InstallerSourceExplicit:
		return 0
	case InstallerSourceExeDir:
		return 1
	case InstallerSourceWorkDir:
		return 2
	case InstallerSourceWorkAssets:
		return 3
	case InstallerSourceExeAssets:
		return 4
	default:
		return 99
	}
}
