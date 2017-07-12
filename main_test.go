package main

import (
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/git"
	"github.com/stretchr/testify/assert"
)

func TestPreCommitCreateReqGraphMarkdown(t *testing.T) {
	err := precommit("/testdata/TestPreCommitCreateReqGraphMarkdown", "/testdata/TestPreCommitCreateReqGraphMarkdown", git.RepoPath()+"/certdocs/attributes.json")
	assert.NotNil(t, err, "Expected some errors but got 0.")

	nLines := strings.Count(err.Error(), "\n")
	assert.Equal(t, 22, nLines, "Number of errors is not correct.")

	assert.Contains(t, err.Error(), "Problems found while parsing")
	assert.Contains(t, err.Error(), "Incorrect requirement type for requirement REQ-TEST-SWH-3. Expected SYS, got SWH.")
	assert.Contains(t, err.Error(), "Incorrect project abbreviation for requirement REQ-TSET-SYS-5. Expected TEST, got TSET.")

	assert.Contains(t, err.Error(), "Invalid requirement sequence number for REQ-TEST-SYS-1, is duplicate.")
	assert.Contains(t, err.Error(), "Invalid requirement sequence number for REQ-TEST-SYS-13: missing requirements in between. Total number of requirements is 10.")

	assert.Contains(t, err.Error(), "Requirement REQ-TEST-SWH-6 in file /testdata/TestPreCommitCreateReqGraphMarkdown/TEST-137-SRD.md has no parents.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-9: REQ-TEST-SYS-3 does not exist.")

	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-4: REQ-TEST-SYS-22 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-8: REQ-TEST-SYS-3 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-2: REQ-TEST-SYS-2 is deleted.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWL-2: REQ-TEST-SYS-2 is deleted.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-5: REQ-TEST-SYS-3 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-10: REQ-TEST-SYS-3 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-11: REQ-TEST-SYS-3 does not exist.")

	assert.Contains(t, err.Error(), "Requirement REQ-TEST-SWH-7 in file /testdata/TestPreCommitCreateReqGraphMarkdown/TEST-137-SRD.md has no parents.")
	assert.Contains(t, err.Error(), "Requirement body must not be empty: REQ-TEST-SWL-4")
	assert.Contains(t, err.Error(), "Requirement number cannot begin with a 0: REQ-TEST-SWL-05. Got 05.")
}

func TestPreCommitCheckReqReferencesMarkdown(t *testing.T) {
	err := precommit("/testdata/TestPreCommitCheckReqReferencesMarkdown", "/testdata/TestPreCommitCheckReqReferencesMarkdown", git.RepoPath()+"/certdocs/attributes.json")
	assert.NotNil(t, err, "Errors expected")

	nLines := strings.Count(err.Error(), "\n")
	assert.Equal(t, 5, nLines, "Number of errors is not correct.")

	assert.Contains(t, err.Error(), "Invalid reference to inexistent requirement REQ-TEST-SYS-22")
	assert.Contains(t, err.Error(), "Invalid reference to deleted requirement REQ-TEST-SYS-2")
	assert.Contains(t, err.Error(), "Requirement 'REQ-TEST-SWH-6' is missing attribute 'Verification'.")
	assert.Contains(t, err.Error(), "Requirement 'REQ-TEST-SWH-8' has invalid value 'gibberish.' in attribute 'VERIFICATION'.")
	assert.Contains(t, err.Error(), "Requirement 'REQ-TEST-SWH-7' is missing attribute 'Safety Impact'.")
}
