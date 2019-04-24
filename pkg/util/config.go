// Package util implements some utility functions.
package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// Config contains configuration for given workspace
type Config struct {
	ContextDir string
	ChartsDir  string
}

// GetConfig tries to find workspace configuration
func GetConfig() (*Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get wd: %v", err)
	}

	suffix := []string{".wk.yaml"}
	for {
		cur := append([]string{wd}, suffix...)
		path := filepath.Clean(filepath.Join(cur...))
		if path == "/.wk.yaml" {
			return nil, fmt.Errorf("config file not found")
		}

		if _, err := os.Stat(path); err == nil {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, err
			}

			c := &Config{}
			if err := yaml.Unmarshal(b, c); err != nil {
				return nil, err
			}
			if c.ChartsDir == "" {
				c.ChartsDir = filepath.Join(filepath.Dir(path), "charts")
			}
			c.ContextDir = filepath.Dir(path)

			return c, nil
		}

		suffix = append([]string{".."}, suffix...)
	}
	return nil, fmt.Errorf("config file not found")
}
