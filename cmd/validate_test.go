package cmd

import (
	"encoding/json"
	"errors"
	"testing"
)

// Capitalized deliberately: files.ValidateManifest produces "Invalid manifest",
// and the fixture mirrors the real message rather than a tidier invention.
var errCheckFailed = errors.New("Invalid manifest") //nolint:staticcheck

// TestExitCodeContract pins the exit codes callers depend on. 0 and 1 are
// produced here; 2 is produced when no manifest can be located, before any
// check runs, and is asserted as a distinct constant so a future edit cannot
// silently collapse it into 1.
func TestExitCodeContract(t *testing.T) {
	t.Parallel()

	if got := exitCodeFor(nil); got != 0 {
		t.Errorf("a passing check should exit 0, got %d", got)
	}

	if got := exitCodeFor(errCheckFailed); got != exitCheckFailed {
		t.Errorf("a failing check should exit %d, got %d", exitCheckFailed, got)
	}

	if exitCheckFailed == exitUsageError {
		t.Error("a rejected manifest and an unrunnable command must not share an exit code")
	}

	if exitCheckFailed != 1 || exitUsageError != 2 {
		t.Errorf("exit codes changed: check=%d usage=%d, want 1 and 2",
			exitCheckFailed, exitUsageError)
	}
}

func TestCheckResultJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		checkErr   error
		wantStatus string
		wantError  string
	}{
		{name: "passing", checkErr: nil, wantStatus: "check-passed"},
		{
			name:       "failing",
			checkErr:   errCheckFailed,
			wantStatus: "check-failed",
			wantError:  "Invalid manifest",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(newCheckResult("amp.yaml", testCase.checkErr))
			if err != nil {
				t.Fatalf("unable to encode: %v", err)
			}

			var decoded map[string]any

			err = json.Unmarshal(encoded, &decoded)
			if err != nil {
				t.Fatalf("output is not valid JSON: %v", err)
			}

			if decoded["status"] != testCase.wantStatus {
				t.Errorf("got status %v, want %q", decoded["status"], testCase.wantStatus)
			}

			if decoded["manifest"] != "amp.yaml" {
				t.Errorf("got manifest %v, want %q", decoded["manifest"], "amp.yaml")
			}

			if testCase.wantError == "" {
				if _, present := decoded["error"]; present {
					t.Error("a passing result must not carry an error field")
				}
			} else if decoded["error"] != testCase.wantError {
				t.Errorf("got error %v, want %q", decoded["error"], testCase.wantError)
			}
		})
	}
}

// TestStatusIsNeverValid guards the wording. These checks are narrower than the
// published manifest schema, so reporting "valid" would overstate them.
func TestStatusIsNeverValid(t *testing.T) {
	t.Parallel()

	for _, status := range []string{statusCheckPassed, statusCheckFailed} {
		if status == "valid" || status == "invalid" {
			t.Errorf("status %q claims more than these checks establish", status)
		}
	}
}
