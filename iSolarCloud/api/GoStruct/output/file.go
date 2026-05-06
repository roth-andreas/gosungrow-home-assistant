package output

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	temp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempName)
		}
	}()

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(filename)
	}
	if err := os.Rename(tempName, filename); err != nil {
		return err
	}
	cleanup = false
	return nil
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

func IsJSONSyntaxError(err error) bool {
	if err == nil {
		return false
	}

	var syntaxErr *json.SyntaxError
	if strings.Contains(strings.ToLower(err.Error()), "unexpected end of json input") {
		return true
	}
	return errors.As(err, &syntaxErr)
}
