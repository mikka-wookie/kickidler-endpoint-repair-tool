package reports

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const SupportBundleName = "kigrepair-support-bundle.zip"

type ArchiveResult struct {
	Path     string   `json:"path"`
	Files    []string `json:"files"`
	Warnings []string `json:"warnings,omitempty"`
}

func CreateSupportBundle(dir string) (ArchiveResult, error) {
	result := ArchiveResult{Path: filepath.Join(dir, SupportBundleName)}
	file, err := os.Create(result.Path)
	if err != nil {
		return result, err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	walkErr := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if samePath(path, result.Path) {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			return nil
		}
		rel = filepath.ToSlash(rel)
		if err := addZipFile(writer, path, rel); err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			return nil
		}
		result.Files = append(result.Files, rel)
		return nil
	})
	if walkErr != nil {
		return result, walkErr
	}
	return result, nil
}

func addZipFile(writer *zip.Writer, path string, name string) error {
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = name
	header.Method = zip.Deflate

	target, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	if _, err := io.Copy(target, source); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
