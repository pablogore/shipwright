package manifest

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// MaxManifestBytes is the hard cap on a manifest's raw byte size before
// decode (design.md D-H threat matrix item 1, tasks.md 4.4). A manifest
// read through Parse/ParseFile that exceeds this size is rejected before
// any YAML decoding happens.
//
// Together with the typed schema (no field capable of holding arbitrary
// depth other than Step.With/Step.When, both bounded further by this raw
// byte cap), this is the deliberate defense against alias-amplification
// ("billion laughs") documents. yaml.v3's own internal alias-ratio
// protection is a backstop only — design.md is explicit that this package
// must not rely on it alone.
//
// 1 MiB is a test constant proposed in design.md, not a contract element
// (design.md Open Questions): tune it if a legitimate manifest ever needs
// more, or if the billion-laughs fixture in security_test.go demands a
// tighter bound.
const MaxManifestBytes = 1 << 20

// ParseFile opens path, applies the MaxManifestBytes cap, decodes with
// KnownFields(true), and runs stages 2-3 validation (ValidateIdentity,
// ValidateStructure). It does not run stages 4-7 (references, graph,
// provider resolution, value binding) — those belong to
// internal/workflow/interp, internal/workflow/graph, and
// internal/workflow/providers.
func ParseFile(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("manifest: open %s: %w", path, err)
	}
	defer f.Close()

	m, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("manifest: %s: %w", path, err)
	}
	return m, nil
}

// Parse decodes and validates (stages 1-3 of the fixed seven-stage
// pipeline, design.md D-H) a manifest read from r. Stage 1 (size-capped
// read + decode) happens first and unconditionally: r's bytes are never
// handed to the YAML decoder directly, only a byte slice already proven to
// be within MaxManifestBytes (see readCapped).
func Parse(r io.Reader) (*Manifest, error) {
	data, err := readCapped(r)
	if err != nil {
		return nil, err
	}

	var m Manifest
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("manifest: decode: %w", err)
	}

	if err := ValidateIdentity(&m); err != nil {
		return nil, err
	}
	if err := ValidateStructure(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

// readCapped reads at most MaxManifestBytes+1 bytes from r via
// io.LimitReader, then rejects the read outright if that extra byte was
// reached. This distinguishes "exactly at the cap" (accepted, subject to
// normal decode/validation) from "over the cap" (rejected with a clear
// error) without ever buffering more than MaxManifestBytes+1 bytes in
// memory, regardless of r's true size — so an oversized manifest is
// rejected, never silently truncated, and never causes an unbounded read.
func readCapped(r io.Reader) ([]byte, error) {
	limited := io.LimitReader(r, MaxManifestBytes+1)

	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("manifest: read: %w", err)
	}

	if len(data) > MaxManifestBytes {
		return nil, fmt.Errorf("manifest: exceeds maximum size of %d bytes", MaxManifestBytes)
	}

	return data, nil
}
