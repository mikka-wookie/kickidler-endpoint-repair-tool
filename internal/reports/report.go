package reports

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"kigrepair/internal/app"
)

type Reporter struct {
	Dir string
}

func New(dir string) (*Reporter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Reporter{Dir: dir}, nil
}

func (r *Reporter) WriteJSON(name string, v any) error {
	return WriteJSON(filepath.Join(r.Dir, ensureExt(name, ".json")), v)
}

func (r *Reporter) WriteText(name string, content string) error {
	return WriteText(filepath.Join(r.Dir, ensureExt(name, ".txt")), content)
}

func (r *Reporter) WriteOperations(results []app.OperationResult) error {
	return WriteJSON(OperationsPath(r.Dir), results)
}

func (r *Reporter) Archive() (string, error) {
	result, err := CreateSupportBundle(r.Dir)
	return result.Path, err
}

func ensureExt(name string, ext string) string {
	if strings.EqualFold(filepath.Ext(name), ext) {
		return name
	}
	return name + ext
}

func marshalJSON(v any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
