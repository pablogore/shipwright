package daggerpin

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/mod/modfile"
)

// daggerJSON mirrors only the fields of dagger.json this guard needs.
type daggerJSON struct {
	EngineVersion string `json:"engineVersion"`
}

// EngineVersion reads dagger.json's engineVersion field.
func EngineVersion(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("daggerpin: read %s: %w", path, err)
	}

	var doc daggerJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("daggerpin: parse %s: %w", path, err)
	}

	if doc.EngineVersion == "" {
		return "", fmt.Errorf("daggerpin: %s has no engineVersion field", path)
	}

	return doc.EngineVersion, nil
}

// GoModDaggerVersion reads the root go.mod's dagger.io/dagger requirement
// version.
func GoModDaggerVersion(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("daggerpin: read %s: %w", path, err)
	}

	f, err := modfile.Parse(path, raw, nil)
	if err != nil {
		return "", fmt.Errorf("daggerpin: parse %s: %w", path, err)
	}

	for _, req := range f.Require {
		if req.Mod.Path == "dagger.io/dagger" {
			return req.Mod.Version, nil
		}
	}

	return "", fmt.Errorf("daggerpin: %s has no dagger.io/dagger requirement", path)
}
