package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/amp-labs/cli/openapi"
)

// ManifestNames are the file names recognized as an integration manifest, in
// the order they are looked for.
var ManifestNames = []string{"amp.yaml", "amp.yml"} //nolint:gochecknoglobals

var ErrManifestNotFound = errors.New("no manifest found")

// FindManifest resolves a user-supplied path to a manifest file. The path may be
// the manifest itself or a directory containing one.
//
// An explicitly named file must still be called amp.yaml or amp.yml, matching
// what `amp deploy` accepts in getZipDir. Accepting arbitrary file names here
// would mean `amp validate` passed on a path that `amp deploy` then rejected,
// which defeats the point of the command.
func FindManifest(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("unable to read %s: %w", path, err)
	}

	if !info.IsDir() {
		if !slices.Contains(ManifestNames, filepath.Base(path)) {
			return "", fmt.Errorf("%w: %s is not a directory nor one of %v",
				ErrManifestNotFound, path, ManifestNames)
		}

		return path, nil
	}

	for _, name := range ManifestNames {
		candidate := filepath.Join(path, name)

		st, err := os.Stat(candidate)
		if err == nil && !st.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("%w: looked for %v in %s", ErrManifestNotFound, ManifestNames, path)
}

// LoadManifest reads and parses a manifest file without validating it.
func LoadManifest(path string) (*openapi.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", path, err)
	}

	return ParseManifest(data)
}
