package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/git"
	"github.com/stretchr/testify/assert"
)

func TestPreCommitCreateReqGraphMarkdown(t *testing.T) {
	d := "testdata/TestPreCommitCreateReqGraphMarkdown"
	err := precommit(d, d, git.RepoPath()+"/certdocs/attributes.json")
	expected := `Incorrect requirement type for requirement REQ-TEST-SWH-3. Expected SYS, got SWH.
Incorrect project abbreviation for requirement REQ-TSET-SYS-5. Expected TEST, got TSET.
Invalid requirement sequence number for REQ-TEST-SYS-1, is duplicate.
Invalid requirement sequence number for REQ-TEST-SYS-13: missing requirements in between. Total number of requirements is 10.
Requirement number cannot begin with a 0: REQ-TEST-SWL-04. Got 04.
Requirement REQ-TEST-SWH-6 in file /testdata/TestPreCommitCreateReqGraphMarkdown/TEST-137-SRD.md has no parents.
Invalid parent of requirement REQ-TEST-SWH-9: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWH-10: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWH-11: REQ-TEST-SYS-3 does not exist.
Requirement REQ-TEST-SWH-7 in file /testdata/TestPreCommitCreateReqGraphMarkdown/TEST-137-SRD.md has no parents.
Invalid parent of requirement REQ-TEST-SWH-8: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWL-2: REQ-TEST-SYS-2 is deleted.
Invalid parent of requirement REQ-TEST-SWH-2: REQ-TEST-SYS-2 is deleted.
Invalid parent of requirement REQ-TEST-SWH-4: REQ-TEST-SYS-22 does not exist.
Invalid parent of requirement REQ-TEST-SWH-5: REQ-TEST-SYS-3 does not exist.`
	checkPrecommitError(t, err, expected)
}

func TestPreCommitCheckReqReferencesMarkdown(t *testing.T) {
	d := "testdata/TestPreCommitCheckReqReferencesMarkdown"
	filepath.Join(git.RepoPath(), d)
	err := precommit(d, d, git.RepoPath()+"/certdocs/attributes.json")
	expected := fmt.Sprintf(`Invalid reference to inexistent requirement REQ-TEST-SYS-22 in %s
Invalid reference to deleted requirement REQ-TEST-SYS-2 in %s
Requirement 'REQ-TEST-SWH-6' is missing attribute 'Verification'.
Requirement 'REQ-TEST-SWH-8' has invalid value 'gibberish.' in attribute 'VERIFICATION'. Expected (Demonstration|Unit [Tt]est|[Tt]est).
Requirement 'REQ-TEST-SWH-7' is missing attribute 'Safety Impact'.`,
		filepath.Join(git.RepoPath(), d, "TEST-137-SRD.md:29"),
		filepath.Join(git.RepoPath(), d, "TEST-137-SRD.md:39"))
	checkPrecommitError(t, err, expected)
}

func checkPrecommitError(t *testing.T, err error, expected string) {
	errs := strings.Split(err.Error(), "\n")
	for _, m := range strings.Split(expected, "\n") {
		found := false
		for i, e := range errs {
			if e == m {
				errs = append(errs[:i], errs[i+1:]...)
				found = true
				break
			}
		}
		assert.Truef(t, found, "Expected error is missing: `%s` from:\n%s", m, err.Error())
	}

	assert.Empty(t, errs, "Got unexpected errors")
}
