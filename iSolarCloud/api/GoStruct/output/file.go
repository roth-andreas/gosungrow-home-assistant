package output

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func ensureParentDir(filename string) error {
	dir := filepath.Dir(filename)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func FileRead(filename string, ref interface{}) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, ref)
}

func FileWrite(filename string, ref interface{}, mode os.FileMode) error {
	data, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return err
	}
	return PlainFileWrite(filename, data, mode)
}

func PlainFileRead(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func PlainFileWrite(filename string, data []byte, mode os.FileMode) error {
	if err := ensureParentDir(filename); err != nil {
		return err
	}
	return os.WriteFile(filename, data, mode)
}

func FileRemove(filename string) error {
	err := os.Remove(filename)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
