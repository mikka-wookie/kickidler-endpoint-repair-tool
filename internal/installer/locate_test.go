package installer

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPreferredInstallerNames(t *testing.T) {
	amd64 := PreferredInstallerNames("amd64")
	wantAMD64 := []string{"grabberEM.x64.msi", "grabberTT.x64.msi", "grabberEM.x32.msi", "grabberTT.x32.msi", "grabber.msi"}
	if !reflect.DeepEqual(amd64, wantAMD64) {
		t.Fatalf("amd64 preference = %#v, want %#v", amd64, wantAMD64)
	}

	x86 := PreferredInstallerNames("386")
	wantX86 := []string{"grabberEM.x32.msi", "grabberTT.x32.msi", "grabberEM.x64.msi", "grabberTT.x64.msi", "grabber.msi"}
	if !reflect.DeepEqual(x86, wantX86) {
		t.Fatalf("386 preference = %#v, want %#v", x86, wantX86)
	}
}

func TestParseInstallerName(t *testing.T) {
	tests := []struct {
		name        string
		wantPackage string
		wantArch    string
		wantOK      bool
	}{
		{name: "grabberEM.x64.msi", wantPackage: "EM", wantArch: "x64", wantOK: true},
		{name: "grabberEM.x32.msi", wantPackage: "EM", wantArch: "x32", wantOK: true},
		{name: "grabberTT.x64.msi", wantPackage: "TT", wantArch: "x64", wantOK: true},
		{name: "grabberTT.x32.msi", wantPackage: "TT", wantArch: "x32", wantOK: true},
		{name: "grabber.msi", wantPackage: "legacy", wantArch: "unknown", wantOK: true},
		{name: "random.msi", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPackage, gotArch, gotOK := ParseInstallerName(tt.name)
			if gotPackage != tt.wantPackage || gotArch != tt.wantArch || gotOK != tt.wantOK {
				t.Fatalf("ParseInstallerName() = %q/%q/%t, want %q/%q/%t", gotPackage, gotArch, gotOK, tt.wantPackage, tt.wantArch, tt.wantOK)
			}
		})
	}
}

func TestResolveInstallerExplicit(t *testing.T) {
	dir := t.TempDir()
	valid := writeFile(t, dir, "custom.msi")
	nonMSI := writeFile(t, dir, "grabber.txt")

	t.Run("valid explicit MSI accepted", func(t *testing.T) {
		resolution, err := ResolveInstallerWithSearchPaths(valid, nil, "amd64")
		if err != nil {
			t.Fatalf("ResolveInstallerWithSearchPaths() error = %v", err)
		}
		if resolution.SelectedPath == "" || resolution.SelectedSource != InstallerSourceExplicit {
			t.Fatalf("selected = %q/%s, want explicit", resolution.SelectedPath, resolution.SelectedSource)
		}
		if resolution.Candidates[0].PackageType != "custom" {
			t.Fatalf("package type = %q, want custom", resolution.Candidates[0].PackageType)
		}
	})

	t.Run("explicit nonexistent rejected", func(t *testing.T) {
		_, err := ResolveInstallerWithSearchPaths(filepath.Join(dir, "missing.msi"), nil, "amd64")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("explicit non-msi rejected", func(t *testing.T) {
		_, err := ResolveInstallerWithSearchPaths(nonMSI, nil, "amd64")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestResolveInstallerAuto(t *testing.T) {
	t.Run("finds installer next to exe", func(t *testing.T) {
		exeDir := t.TempDir()
		writeFile(t, exeDir, "grabberEM.x64.msi")
		resolution := resolveAuto(t, "amd64", []InstallerSearchPath{{Dir: exeDir, Source: InstallerSourceExeDir}})
		assertSelected(t, resolution, "grabberEM.x64.msi", InstallerSourceExeDir)
	})

	t.Run("finds installer in workdir", func(t *testing.T) {
		workDir := t.TempDir()
		writeFile(t, workDir, "grabberTT.x64.msi")
		resolution := resolveAuto(t, "amd64", []InstallerSearchPath{{Dir: workDir, Source: InstallerSourceWorkDir}})
		assertSelected(t, resolution, "grabberTT.x64.msi", InstallerSourceWorkDir)
	})

	t.Run("prefers exe-dir over workdir for same preferred name", func(t *testing.T) {
		exeDir := t.TempDir()
		workDir := t.TempDir()
		writeFile(t, exeDir, "grabberEM.x64.msi")
		writeFile(t, workDir, "grabberEM.x64.msi")
		resolution := resolveAuto(t, "amd64", []InstallerSearchPath{
			{Dir: exeDir, Source: InstallerSourceExeDir},
			{Dir: workDir, Source: InstallerSourceWorkDir},
		})
		assertSelected(t, resolution, "grabberEM.x64.msi", InstallerSourceExeDir)
	})

	t.Run("prefers x64 over x32 on amd64", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "grabberEM.x32.msi")
		writeFile(t, dir, "grabberEM.x64.msi")
		resolution := resolveAuto(t, "amd64", []InstallerSearchPath{{Dir: dir, Source: InstallerSourceExeDir}})
		assertSelected(t, resolution, "grabberEM.x64.msi", InstallerSourceExeDir)
	})

	t.Run("prefers EM x64 over TT x64 on amd64", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "grabberTT.x64.msi")
		writeFile(t, dir, "grabberEM.x64.msi")
		resolution := resolveAuto(t, "amd64", []InstallerSearchPath{{Dir: dir, Source: InstallerSourceExeDir}})
		assertSelected(t, resolution, "grabberEM.x64.msi", InstallerSourceExeDir)
	})

	t.Run("prefers x32 over x64 on 386", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "grabberEM.x64.msi")
		writeFile(t, dir, "grabberEM.x32.msi")
		resolution := resolveAuto(t, "386", []InstallerSearchPath{{Dir: dir, Source: InstallerSourceExeDir}})
		assertSelected(t, resolution, "grabberEM.x32.msi", InstallerSourceExeDir)
	})

	t.Run("returns clear error when none found", func(t *testing.T) {
		_, err := ResolveInstallerWithSearchPaths("", []InstallerSearchPath{{Dir: t.TempDir(), Source: InstallerSourceExeDir}}, "amd64")
		if !errors.Is(err, ErrInstallerNotFound) {
			t.Fatalf("error = %v, want ErrInstallerNotFound", err)
		}
	})
}

func resolveAuto(t *testing.T, arch string, paths []InstallerSearchPath) InstallerResolution {
	t.Helper()
	resolution, err := ResolveInstallerWithSearchPaths("", paths, arch)
	if err != nil {
		t.Fatalf("ResolveInstallerWithSearchPaths() error = %v", err)
	}
	return resolution
}

func assertSelected(t *testing.T, resolution InstallerResolution, name string, source InstallerSource) {
	t.Helper()
	if filepath.Base(resolution.SelectedPath) != name || resolution.SelectedSource != source {
		t.Fatalf("selected = %q/%s, want %s/%s", resolution.SelectedPath, resolution.SelectedSource, name, source)
	}
	if len(resolution.Candidates) == 0 {
		t.Fatal("expected discovered candidates")
	}
}

func writeFile(t *testing.T, dir string, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("msi"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
