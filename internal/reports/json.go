package reports

import "os"

func WriteJSON(path string, v any) error {
	data, err := marshalJSON(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
