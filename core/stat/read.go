package stat

import (
	"os"
)

func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func ReadString(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
