package files

import (
	"os"
	"path/filepath"
	"testing"
)

const validManifest = `specVersion: 1.0.0
integrations:
  - name: test-integration
    provider: hubspot
    read:
      objects:
        - objectName: contacts
          destination: webhook
          schedule: "0 * * * *"
`

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()

	path := filepath.Join(dir, name)

	err := os.WriteFile(path, []byte(body), 0o600)
	if err != nil {
		t.Fatalf("unable to write fixture: %v", err)
	}

	return path
}

func TestFindManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		create  []string
		lookup  string
		want    string
		wantErr bool
	}{
		{name: "amp.yaml in a directory", create: []string{"amp.yaml"}, want: "amp.yaml"},
		{name: "amp.yml in a directory", create: []string{"amp.yml"}, want: "amp.yml"},
		{
			name:   "amp.yaml wins when both exist",
			create: []string{"amp.yaml", "amp.yml"},
			want:   "amp.yaml",
		},
		{
			name:   "an explicitly named amp.yaml is used as given",
			create: []string{"amp.yaml"},
			lookup: "amp.yaml",
			want:   "amp.yaml",
		},
		{
			// amp deploy's getZipDir rejects explicit files not named amp.yaml
			// or amp.yml, so accepting one here would let validate pass a path
			// that deploy refuses.
			name:    "an explicit file with another name is rejected, as amp deploy does",
			create:  []string{"custom.yaml"},
			lookup:  "custom.yaml",
			wantErr: true,
		},
		{name: "no manifest present", create: nil, wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for _, name := range testCase.create {
				writeFile(t, dir, name, validManifest)
			}

			lookup := dir
			if testCase.lookup != "" {
				lookup = filepath.Join(dir, testCase.lookup)
			}

			got, err := FindManifest(lookup)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("got %q, want an error", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != filepath.Join(dir, testCase.want) {
				t.Errorf("got %q, want %q", got, filepath.Join(dir, testCase.want))
			}
		})
	}
}

func TestLoadManifest(t *testing.T) {
	t.Parallel()

	path := writeFile(t, t.TempDir(), "amp.yaml", validManifest)

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Integrations) != 1 {
		t.Fatalf("got %d integration(s), want 1", len(manifest.Integrations))
	}
}

func TestLoadManifestMissingFile(t *testing.T) {
	t.Parallel()

	_, err := LoadManifest(filepath.Join(t.TempDir(), "absent.yaml"))
	if err == nil {
		t.Fatal("want an error for a missing file")
	}
}

// TestValidateManifestKnownGaps records what these checks do NOT catch. The
// command's help and status wording depend on these staying true; if one of
// these starts failing, the check got stricter and the wording should follow.
func TestValidateManifestKnownGaps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "schedule is required by the published schema but not enforced here",
			body: `specVersion: 1.0.0
integrations:
  - name: test-integration
    provider: hubspot
    read:
      objects:
        - objectName: contacts
          destination: webhook
`,
		},
		{
			name: "duplicate keys are accepted, and the last value silently wins",
			body: `specVersion: 1.0.0
integrations:
  - name: test-integration
    provider: hubspot
    read:
      objects:
        - objectName: contacts
          destination: first
          destination: second
          schedule: "0 * * * *"
`,
		},
		{
			name: "unknown properties are accepted",
			body: `specVersion: 1.0.0
integrations:
  - name: test-integration
    provider: hubspot
    notAField: true
    read:
      objects:
        - objectName: contacts
          destination: webhook
          schedule: "0 * * * *"
`,
		},
		{
			name: "a provider that does not exist is accepted",
			body: `specVersion: 1.0.0
integrations:
  - name: test-integration
    provider: NotARealProvider
    read:
      objects:
        - objectName: contacts
          destination: webhook
          schedule: "0 * * * *"
`,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := writeFile(t, t.TempDir(), "amp.yaml", testCase.body)

			manifest, err := LoadManifest(path)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			err = ValidateManifest(manifest)
			if err != nil {
				t.Fatalf("this gap appears to have been closed, which is good, "+
					"but the validate command's wording assumes it is open: %v", err)
			}
		})
	}
}

func TestValidateManifestRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing destination",
			body: `specVersion: 1.0.0
integrations:
  - name: test-integration
    provider: hubspot
    read:
      objects:
        - objectName: contacts
          schedule: "0 * * * *"
`,
		},
		{
			name: "missing objectName",
			body: `specVersion: 1.0.0
integrations:
  - name: test-integration
    provider: hubspot
    read:
      objects:
        - destination: webhook
          schedule: "0 * * * *"
`,
		},
		{
			name: "empty objects list",
			body: `specVersion: 1.0.0
integrations:
  - name: test-integration
    provider: hubspot
    read:
      objects: []
`,
		},
		{
			name: "unsupported spec version",
			body: `specVersion: 2.0.0
integrations:
  - name: test-integration
    provider: hubspot
`,
		},
		{
			name: "no integrations",
			body: `specVersion: 1.0.0
integrations: []
`,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := writeFile(t, t.TempDir(), "amp.yaml", testCase.body)

			manifest, err := LoadManifest(path)
			if err != nil {
				return // a parse failure is also a rejection
			}

			err = ValidateManifest(manifest)
			if err == nil {
				t.Fatal("want this manifest to be rejected")
			}
		})
	}
}
