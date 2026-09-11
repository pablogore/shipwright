// Package toolchains is the id -> core.Toolchain registry (issue #284,
// design.md's "Registry extensibility" requirement): adding a new
// toolchain needs one new provider file (satisfying the existing
// shipwright.RuntimeUpgrader), one new descriptor package (satisfying
// core.Toolchain), and a one-line addition to Registry below — zero
// edits to internal/toolbump/core or the delivery script. Design.md's
// Paper Walkthrough proves this by construction for a hypothetical Java
// toolchain; this file is exactly the "+1 line" step it describes.
package toolchains

import (
	"github.com/pablogore/shipwright/internal/toolbump/core"
	gotoolchain "github.com/pablogore/shipwright/internal/toolbump/toolchains/go"
)

// Registry maps a Toolchain's ID() to its descriptor. The future CLI
// (Phase 3) selects one of these by its -toolchain flag value;
// core.Run itself never looks at this map — it receives an
// already-resolved core.Toolchain, so Registry is purely a CLI-facing
// lookup table.
var Registry = map[string]core.Toolchain{
	gotoolchain.Toolchain.ID(): gotoolchain.Toolchain,
}
