package shipwright

// ContractVersion is the source of truth for the public capability
// contract's SemVer-style compatibility guarantee (design.md D-E). It
// resolves independently of the CLI binary's release SemVer (goreleaser +
// CHANGELOG.md, main.go's `Version` var) and of the `dagger.json`
// engineVersion pin — three separate version axes that are never conflated.
//
// A breaking change to the guaranteed surface (the five capability
// interfaces, the Shipwright/composition-type surface, and the
// pkg/shipwright config structs) MUST bump the major segment here and ship
// a written migration note. Internal, non-exported packages carry no
// compatibility guarantee and never require a bump.
const ContractVersion = "1.0.0"
