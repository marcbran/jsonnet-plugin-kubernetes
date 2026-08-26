package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Getter interface {
	Get(ctx context.Context, path string) ([]byte, error)
}

type Input struct {
	OutDir string
}

type Output struct {
	OutDir        string
	BundleFile    string
	GroupVersions []string
}

var groupVersionKey = regexp.MustCompile(`^(api/v[0-9][^/]*|apis/[^/]+/v[0-9][^/]*)$`)

var sanitizeChars = regexp.MustCompile(`[^A-Za-z0-9_.-]`)

type indexDocument struct {
	Paths map[string]struct {
		ServerRelativeURL string `json:"serverRelativeURL"`
	} `json:"paths"`
}

func Exec(ctx context.Context, getters []Getter, in Input) (Output, error) {
	if len(getters) == 0 {
		return Output{}, fmt.Errorf("at least one context is required")
	}
	if err := os.MkdirAll(in.OutDir, 0755); err != nil {
		return Output{}, err
	}

	bundle := map[string]json.RawMessage{}
	mergedIndex := map[string]any{}

	for _, getter := range getters {
		indexBody, err := getter.Get(ctx, "/openapi/v3")
		if err != nil {
			return Output{}, fmt.Errorf("fetch openapi/v3 index: %w", err)
		}
		var index indexDocument
		if err := json.Unmarshal(indexBody, &index); err != nil {
			return Output{}, fmt.Errorf("parse openapi/v3 index: %w", err)
		}

		for key, entry := range index.Paths {
			if !groupVersionKey.MatchString(key) {
				continue
			}
			if _, seen := bundle[key]; seen {
				continue
			}
			specBody, err := getter.Get(ctx, entry.ServerRelativeURL)
			if err != nil {
				return Output{}, fmt.Errorf("fetch %s: %w", key, err)
			}
			dir := filepath.Join(in.OutDir, sanitizeName(key))
			if err := os.MkdirAll(dir, 0755); err != nil {
				return Output{}, err
			}
			if err := os.WriteFile(filepath.Join(dir, "spec.json"), specBody, 0644); err != nil {
				return Output{}, err
			}
			bundle[key] = json.RawMessage(specBody)
			mergedIndex[key] = map[string]any{"serverRelativeURL": entry.ServerRelativeURL}
		}
	}

	keys := make([]string, 0, len(bundle))
	for key := range bundle {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	indexBody, err := json.MarshalIndent(map[string]any{"paths": mergedIndex}, "", "  ")
	if err != nil {
		return Output{}, err
	}
	indexBody = append(indexBody, '\n')
	if err := os.WriteFile(filepath.Join(in.OutDir, "index.json"), indexBody, 0644); err != nil {
		return Output{}, err
	}

	bundleFile := filepath.Join(in.OutDir, "specs.json")
	bundleBody, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return Output{}, err
	}
	bundleBody = append(bundleBody, '\n')
	if err := os.WriteFile(bundleFile, bundleBody, 0644); err != nil {
		return Output{}, err
	}

	return Output{
		OutDir:        in.OutDir,
		BundleFile:    bundleFile,
		GroupVersions: keys,
	}, nil
}

func sanitizeName(key string) string {
	return sanitizeChars.ReplaceAllString(strings.ReplaceAll(key, "/", "__"), "_")
}
