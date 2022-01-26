package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/git"
	"github.com/stretchr/testify/assert"
)

func RunValidate(t *testing.T, certdocpath, codepath, schemapath string) (string, error) {
	// set the test paths
	*fCertdocPath = certdocpath
	*fCodePath = codepath
	*fSchemaPath = schemapath
	// prepare capture of stdout
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	// run the command
	err := validate()
	assert.Empty(t, err, "Got unexpected error")
	// save stdout data and reset
	w.Close()
	buf, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	return string(buf), err
}

func TestValidateCreateReqGraphMarkdown(t *testing.T) {
	actual, err := RunValidate(t, "testdata/TestValidateCreateReqGraphMarkdown", "testdata/TestValidateCreateReqGraphMarkdown", git.RepoPath()+"/certdocs/attributes.json")
	assert.Empty(t, err, "Got unexpected error")

	expected := `Incorrect requirement type for requirement REQ-TEST-SWH-3. Expected SYS, got SWH.
Incorrect project abbreviation for requirement REQ-TSET-SYS-5. Expected TEST, got TSET.
Invalid requirement sequence number for REQ-TEST-SYS-1, is duplicate.
Invalid requirement sequence number for REQ-TEST-SYS-13: missing requirements in between. Expected ID Number 9.
Requirement number cannot begin with a 0: REQ-TEST-SWL-04. Got 04.
Requirement REQ-TEST-SWH-6 in file /testdata/TestValidateCreateReqGraphMarkdown/TEST-137-SRD.md has no parents.
Invalid parent of requirement REQ-TEST-SWH-9: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWH-10: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWH-11: REQ-TEST-SYS-3 does not exist.
Requirement REQ-TEST-SWH-7 in file /testdata/TestValidateCreateReqGraphMarkdown/TEST-137-SRD.md has no parents.
Invalid parent of requirement REQ-TEST-SWH-8: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWL-2: REQ-TEST-SYS-2 is deleted.
Invalid parent of requirement REQ-TEST-SWH-2: REQ-TEST-SYS-2 is deleted.
Invalid parent of requirement REQ-TEST-SWH-4: REQ-TEST-SYS-22 does not exist.
Invalid parent of requirement REQ-TEST-SWH-5: REQ-TEST-SYS-3 does not exist.
Invalid reference to deleted requirement REQ-TEST-SYS-2 in body of REQ-TEST-SWH-11.
Invalid reference to non existent requirement REQ-TEST-SYS-22 in body of REQ-TEST-SWH-5.
Requirement 'REQ-TEST-SWH-7' is missing attribute 'Parents'.
Requirement 'REQ-TEST-SWH-9' is missing attribute 'Safety Impact'.
Requirement 'REQ-TEST-SWH-6' is missing attribute 'Parents'.
Requirement 'REQ-TEST-SWH-8' is missing attribute 'Verification'.
Requirement 'REQ-TEST-SWH-10' has invalid value 'None.' in attribute 'VERIFICATION'. Expected (Demonstration|Unit [Tt]est|[Tt]est).
Requirement 'REQ-TEST-SWH-10' is missing attribute 'Safety Impact'.
WARNING. Validation failed`

	checkValidateError(t, actual, expected)
}

func TestValidateCheckReqReferencesMarkdown(t *testing.T) {
	actual, err := RunValidate(t, "testdata/TestValidateCheckReqReferencesMarkdown", "testdata/TestValidateCheckReqReferencesMarkdown", git.RepoPath()+"/certdocs/attributes.json")
	assert.Empty(t, err, "Got unexpected error")

	expected := `Invalid reference to non existent requirement REQ-TEST-SYS-22 in body of REQ-TEST-SWH-3.
Invalid reference to deleted requirement REQ-TEST-SYS-2 in body of REQ-TEST-SWH-4.
Requirement 'REQ-TEST-SWH-6' is missing attribute 'Verification'.
Requirement 'REQ-TEST-SWH-8' has invalid value 'gibberish.' in attribute 'VERIFICATION'. Expected (Demonstration|Unit [Tt]est|[Tt]est).
Requirement 'REQ-TEST-SWH-7' is missing attribute 'Safety Impact'.
WARNING. Validation failed`

	checkValidateError(t, actual, expected)
}

func checkValidateError(t *testing.T, validate_errors string, expected string) {
	errs := strings.Split(validate_errors, "\n")
	for i, e := range errs {
		if e == "" {
			errs = append(errs[:i], errs[i+1:]...)
		}
	}
	for _, m := range strings.Split(expected, "\n") {
		found := false
		for i, e := range errs {
			if e == m {
				errs = append(errs[:i], errs[i+1:]...)
				found = true
				break
			}
		}
		assert.Truef(t, found, "Expected error is missing: `%s` from:\n%s", m, validate_errors)
	}

	assert.Empty(t, errs, "Got unexpected errors")
}
