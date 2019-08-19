package helper

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

func GetExecutableFilePath() (string, error) {
	executableFilePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executableFilePath), nil
}

func ParseStdout(cmdOut []byte) (map[string]interface{}, error) {
	var raw map[string]interface{}
	err := json.Unmarshal(cmdOut, &raw)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to parse SSH command stdout")
	}

	return raw, nil
}