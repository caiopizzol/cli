package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/amp-labs/cli/files"
	"github.com/amp-labs/cli/logger"
	"github.com/spf13/cobra"
)

// Exit codes, which are a contract for scripts and CI: 1 means the manifest was
// rejected, 2 means the command could not run. They must stay distinct, so this
// command does not use logger.Fatal, which always exits 1.
const (
	exitCheckFailed = 1
	exitUsageError  = 2
)

// Status values. Deliberately not "valid": these are the same local checks that
// `amp deploy` runs, which do not cover everything the published manifest schema
// requires. See the note on `schedule` in the command's long help.
const (
	statusCheckPassed = "check-passed"
	statusCheckFailed = "check-failed"
)

// checkResult is the --json output shape.
type checkResult struct {
	Status   string `json:"status"`
	Manifest string `json:"manifest"`
	Error    string `json:"error,omitempty"`
}

var (
	validateJSON bool //nolint:gochecknoglobals

	validateCmd = &cobra.Command{ //nolint:gochecknoglobals
		Use:   "validate [ampYamlSourcePath]",
		Short: "Run amp deploy's local manifest checks without deploying",
		Long: `Run the same local manifest checks that amp deploy performs, without uploading
or changing any remote state.

Requires neither login nor a project.

The path may be a manifest file or a directory containing amp.yaml or amp.yml,
and defaults to the current directory.

Exit codes: 0 the checks passed, 1 the manifest was rejected, 2 the command
could not run.

This is a local check, not full validation. It does not verify provider object
names, whether the referenced destination exists, OAuth scopes, or anything the
server enforces at deploy time. It is also narrower than the published manifest
schema in at least one known way: the schema marks 'schedule' required on read
objects, and these checks do not enforce it. Passing therefore means "amp deploy
would not reject this locally", which is why the status is reported as
check-passed rather than valid.`,
		Example: `  amp validate
  amp validate ./integrations
  amp validate amp.yaml --json`,
		Args: cobra.MaximumNArgs(1),
		Run:  runValidate,
	}
)

func init() {
	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "Emit machine-readable JSON")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) {
	source := "."
	if len(args) > 0 {
		source = args[0]
	}

	manifestPath, err := files.FindManifest(source)
	if err != nil {
		// There is nothing to check, which is a usage problem rather than a
		// rejected manifest. The two must not share an exit code.
		fmt.Fprintf(os.Stderr, "amp validate: %v\n", err)
		os.Exit(exitUsageError)
	}

	err = checkManifest(manifestPath)

	if validateJSON {
		emitCheckJSON(newCheckResult(manifestPath, err))
	} else {
		emitCheckText(manifestPath, err)
	}

	os.Exit(exitCodeFor(err))
}

// newCheckResult builds the --json payload. Split out from emission so the
// output contract can be tested without running the binary.
func newCheckResult(manifestPath string, checkErr error) checkResult {
	result := checkResult{Status: statusCheckPassed, Manifest: manifestPath}
	if checkErr != nil {
		result.Status = statusCheckFailed
		result.Error = checkErr.Error()
	}

	return result
}

// exitCodeFor maps a check outcome to a process exit code. Callers rely on 0, 1
// and 2 being distinct, so this is covered by tests rather than left implicit.
func exitCodeFor(checkErr error) int {
	if checkErr != nil {
		return exitCheckFailed
	}

	return 0
}

func checkManifest(manifestPath string) error {
	manifest, err := files.LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	return files.ValidateManifest(manifest)
}

func emitCheckJSON(result checkResult) {
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.FatalErr("unable to encode the result", err)
	}

	fmt.Fprintln(os.Stdout, string(encoded))
}

func emitCheckText(manifestPath string, checkErr error) {
	if checkErr != nil {
		logger.Infof("%s was rejected", manifestPath)
		logger.Info(checkErr.Error())

		return
	}

	logger.Infof("%s passed amp deploy's local checks", manifestPath)
	logger.Info("This is not full validation: object names, destinations, scopes, " +
		"and server-side checks are not covered.")
}
