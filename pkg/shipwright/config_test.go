package shipwright_test

import (
	"reflect"
	"sort"
	"testing"

	"dagger.io/dagger"

	"github.com/pablogore/shipwright/pkg/shipwright"
)

// exportedFieldNames returns the sorted, exported field names of a struct
// type, so a config struct's exact field set can be asserted without
// depending on declaration order.
func exportedFieldNames(t reflect.Type) []string {
	names := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.IsExported() {
			names = append(names, f.Name)
		}
	}
	sort.Strings(names)
	return names
}

// TestConfigStructs_ScopedToOwnCapability asserts each decomposed config
// struct (design.md D-D) carries exactly the fields owned by its own
// capability — never a field belonging to a sibling capability. This is the
// compiler-enforced orthogonality D-D exists to guarantee: SourceConfig has
// no Build-only field, ArtifactConfig has no Source-only field, and so on.
func TestConfigStructs_ScopedToOwnCapability(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		typ    reflect.Type
		fields []string
	}{
		{
			name: "SourceConfig",
			typ:  reflect.TypeFor[shipwright.SourceConfig](),
			fields: []string{
				"GitProtocol", "GitRef", "GitRepo", "GitUserEmail", "GitUserName", "SSHPrivateKey",
			},
		},
		{
			name: "BuildConfig",
			typ:  reflect.TypeFor[shipwright.BuildConfig](),
			fields: []string{
				"BinaryName", "BuildMode", "GoVersion", "JavaVersion",
			},
		},
		{
			name: "TestConfig",
			typ:  reflect.TypeFor[shipwright.TestConfig](),
			fields: []string{
				"Coverage",
			},
		},
		{
			name: "ArtifactConfig",
			typ:  reflect.TypeFor[shipwright.ArtifactConfig](),
			fields: []string{
				"BranchName", "BuildTag", "CommitSHA", "ImageName", "ImageTag",
				"Registry", "RegistryPass", "RegistryToken", "RegistryURL",
				"RegistryUser", "Token", "Version",
			},
		},
		{
			name:   "DeployConfig",
			typ:    reflect.TypeFor[shipwright.DeployConfig](),
			fields: []string{},
		},
		{
			name:   "RunConfig",
			typ:    reflect.TypeFor[shipwright.RunConfig](),
			fields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.typ.Kind() != reflect.Struct {
				t.Fatalf("shipwright.%s is not a struct type", tt.name)
			}

			want := append([]string{}, tt.fields...)
			sort.Strings(want)
			got := exportedFieldNames(tt.typ)

			if !reflect.DeepEqual(got, want) {
				t.Fatalf("shipwright.%s field set = %v, want %v", tt.name, got, want)
			}
		})
	}
}

// TestConfigStructs_DoNotCarryLiveDaggerHandles asserts the dropped fields
// from design.md D-D (Image, ImageContainer, ImageRef — runtime state, not
// configuration) never reappear on ArtifactConfig.
func TestConfigStructs_DoNotCarryLiveDaggerHandles(t *testing.T) {
	t.Parallel()

	dropped := []string{"Image", "ImageContainer", "ImageRef"}
	got := exportedFieldNames(reflect.TypeFor[shipwright.ArtifactConfig]())

	for _, name := range dropped {
		for _, g := range got {
			if g == name {
				t.Fatalf("shipwright.ArtifactConfig must not carry dropped runtime-state field %q (design.md D-D)", name)
			}
		}
	}
}

// TestArtifactConfig_CredentialFieldsAreSecretTyped is the security-relevant
// RED/GREEN task (1.5): RegistryPass, RegistryToken, and Token MUST be
// *dagger.Secret, never a plaintext string, on the public config surface.
func TestArtifactConfig_CredentialFieldsAreSecretTyped(t *testing.T) {
	t.Parallel()

	secretType := reflect.TypeFor[*dagger.Secret]()
	artifactConfigType := reflect.TypeFor[shipwright.ArtifactConfig]()

	credentialFields := []string{"RegistryPass", "RegistryToken", "Token"}

	for _, name := range credentialFields {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			field, ok := artifactConfigType.FieldByName(name)
			if !ok {
				t.Fatalf("shipwright.ArtifactConfig has no field %q", name)
			}

			if field.Type != secretType {
				t.Fatalf("shipwright.ArtifactConfig.%s type = %s, want %s (credentials MUST cross the public contract as *dagger.Secret, never string)",
					name, field.Type, secretType)
			}
		})
	}
}
