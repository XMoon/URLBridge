package bridge

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ConfigLookup struct {
	Path       string
	Found      bool
	Candidates []string
	Explicit   bool
}

func LookupConfigPath(explicitPath, executablePath, fileName string, defaultPaths []string) (ConfigLookup, error) {
	if explicitPath != "" {
		path, err := normalizePath(explicitPath)
		if err != nil {
			return ConfigLookup{}, fmt.Errorf("resolve config path: %w", err)
		}

		return ConfigLookup{
			Path:       path,
			Found:      fileExists(path),
			Candidates: []string{path},
			Explicit:   true,
		}, nil
	}

	candidates, err := ConfigCandidatePaths(executablePath, fileName, defaultPaths)
	if err != nil {
		return ConfigLookup{}, err
	}

	lookup := ConfigLookup{Candidates: candidates}
	for _, candidate := range candidates {
		if fileExists(candidate) {
			lookup.Path = candidate
			lookup.Found = true
			break
		}
	}

	return lookup, nil
}

func ConfigCandidatePaths(executablePath, fileName string, defaultPaths []string) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve current working directory: %w", err)
	}

	return ConfigCandidatePathsFrom(cwd, executablePath, fileName, defaultPaths)
}

func ConfigCandidatePathsFrom(cwd, executablePath, fileName string, defaultPaths []string) ([]string, error) {
	if fileName == "" {
		return nil, fmt.Errorf("config file name is required")
	}

	var candidates []string
	if cwd != "" {
		candidates = append(candidates, filepath.Join(cwd, fileName))
	}
	if executablePath != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(executablePath), fileName))
	}
	candidates = append(candidates, defaultPaths...)

	seen := make(map[string]struct{}, len(candidates))
	normalized := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		path, err := normalizePath(candidate)
		if err != nil {
			return nil, fmt.Errorf("resolve config path %q: %w", candidate, err)
		}
		if _, exists := seen[path]; exists {
			continue
		}

		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}

	return normalized, nil
}

func DecodeYAMLFile(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	return nil
}

func EncodeYAML(v any) ([]byte, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal yaml: %w", err)
	}

	return append(data, '\n'), nil
}

func normalizePath(path string) (string, error) {
	return filepath.Abs(path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
