package util

import (
	"fmt"
	"io/ioutil"
)

func WriteTempFile(data []byte) (string, error) {
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		return "", fmt.Errorf("could create temp file: %v", err)
	}

	if _, err := tmpfile.Write(data); err != nil {
		return "", fmt.Errorf("could write temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("could close temp file: %v", err)
	}
	return tmpfile.Name(), nil
}
