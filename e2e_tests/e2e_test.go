//go:build integration

package e2etest

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/vm-compat/analyzer"
	"github.com/stretchr/testify/assert"
)

const testdataDir = "testdata"

type testcase struct {
	path      string
	isPassing bool
}

func runTest(t *testing.T, vmProfile string, cases map[string]testcase) {
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command("../bin/vm-compact", "analyze", "-vm-profile", vmProfile, "-format", "json", tc.path)

			var out bytes.Buffer
			var errOut bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &errOut
			err := cmd.Run()
			if err != nil {
				t.Fatalf("Failed to run CLI: %v. errorOutput: %s", err, errOut.String())
			}

			issues := []*analyzer.Issue{}
			json.Unmarshal(out.Bytes(), &issues)

			if tc.isPassing {
				for i := range issues {
					assert.NotEqual(t, analyzer.IssueSeverityCritical, issues[i].Severity, "Found Critical issue")
				}
			} else {
				var criticalIssueFound bool
				for i := range issues {
					if issues[i].Severity == analyzer.IssueSeverityCritical {
						criticalIssueFound = true
					}
				}

				assert.True(t, criticalIssueFound, "No critical issues found")
			}
		})
	}
}

func TestMips(t *testing.T) {
	cases := map[string]testcase{
		"hello_world": {
			path:      filepath.Join(testdataDir, "hello"),
			isPassing: true,
		},
		"sys-clockgettime": {
			path: filepath.Join(testdataDir, "sys-clockgettime"),
		},
		"sys-getrandom": {
			path: filepath.Join(testdataDir, "sys-getrandom"),
		},
	}
	runTest(t, "../profile/cannon/cannon-32.yaml", cases)
}

func TestMips64(t *testing.T) {
	cases := map[string]testcase{
		"hello_world": {
			path:      filepath.Join(testdataDir, "hello"),
			isPassing: true,
		},
		"sys-clockgettime": {
			path: filepath.Join(testdataDir, "sys-clockgettime"),
		},
		"sys-getrandom": {
			path: filepath.Join(testdataDir, "sys-getrandom"),
		},
	}
	runTest(t, "../profile/cannon/cannon-64.yaml", cases)
}
