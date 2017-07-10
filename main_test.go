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
	assert.Equal(t, 21, nLines, "Number of errors is not correct.")

	assert.Contains(t, err.Error(), "Problems found while parsing")
	assert.Contains(t, err.Error(), "Incorrect requirement type for requirement REQ-TEST-SWH-003. Expected SYS, got SWH.")
	assert.Contains(t, err.Error(), "Incorrect project abbreviation for requirement REQ-TSET-SYS-005. Expected TEST, got TSET.")

	assert.Contains(t, err.Error(), "Invalid requirement sequence number for REQ-TEST-SYS-001, is duplicate.")
	assert.Contains(t, err.Error(), "Invalid requirement sequence number for REQ-TEST-SYS-013: missing requirements in between. Total number of requirements is 10.")

	assert.Contains(t, err.Error(), "Requirement REQ-TEST-SWH-006 in file /testdata/TestPreCommitCreateReqGraphMarkdown/TEST-137-SRD.md has no parents.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-009: REQ-TEST-SYS-003 does not exist.")

	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-004: REQ-TEST-SYS-022 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-008: REQ-TEST-SYS-003 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-002: REQ-TEST-SYS-002 is deleted.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWL-002: REQ-TEST-SYS-002 is deleted.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-005: REQ-TEST-SYS-003 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-010: REQ-TEST-SYS-003 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-TEST-SWH-011: REQ-TEST-SYS-003 does not exist.")

	assert.Contains(t, err.Error(), "Requirement REQ-TEST-SWH-007 in file /testdata/TestPreCommitCreateReqGraphMarkdown/TEST-137-SRD.md has no parents.")
	assert.Contains(t, err.Error(), "Requirement body must not be empty: REQ-TEST-SWL-004")
}

func TestPreCommitCheckReqReferencesMarkdown(t *testing.T) {
	err := precommit("/testdata/TestPreCommitCheckReqReferencesMarkdown", "/testdata/TestPreCommitCheckReqReferencesMarkdown", git.RepoPath()+"/certdocs/attributes.json")
	assert.NotNil(t, err, "Errors expected")

	nLines := strings.Count(err.Error(), "\n")
	assert.Equal(t, 5, nLines, "Number of errors is not correct.")

	assert.Contains(t, err.Error(), "Invalid reference to inexistent requirement REQ-TEST-SYS-022")
	assert.Contains(t, err.Error(), "Invalid reference to deleted requirement REQ-TEST-SYS-002")
	assert.Contains(t, err.Error(), "Requirement 'REQ-TEST-SWH-006' is missing attribute 'Verification'.")
	assert.Contains(t, err.Error(), "Requirement 'REQ-TEST-SWH-008' has invalid value 'gibberish.' in attribute 'VERIFICATION'.")
	assert.Contains(t, err.Error(), "Requirement 'REQ-TEST-SWH-007' is missing attribute 'Safety Impact'.")
}
