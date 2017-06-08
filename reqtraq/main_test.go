package main

import (
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/git"

	"github.com/stretchr/testify/assert"
)

func TestPreCommitCreateReqGraph(t *testing.T) {
	err := precommit(git.RepoPath()+"/reqtraq/testdata/TestPreCommitCreateReqGraph", git.RepoPath()+"/reqtraq/testdata/TestPreCommitCreateReqGraph", git.RepoPath()+"/certdocs/attributes.json")
	assert.NotNil(t, err, "Expected 21 errors, got 0.")

	nLines := strings.Count(err.Error(), "\n")
	assert.Equal(t, 23, nLines, "Number of errors is not correct.")

	assert.Contains(t, err.Error(), "Problems found while parsing")
	assert.Contains(t, err.Error(), "Incorrect requirement type for requirement REQ-0-TEST-SWH-003. Expected SYS, got SWH.")
	assert.Contains(t, err.Error(), "Incorrect project ID for requirement REQ-1-TEST-SYS-004. Expected 0, got 1.")
	assert.Contains(t, err.Error(), "Incorrect project abbreviation for requirement REQ-0-TSET-SYS-005. Expected TEST, got TSET.")

	assert.Contains(t, err.Error(), "malformed requirement: missing ID in first 40 characters: \"\\nREG-0-TEST-SYS-006")
	assert.Contains(t, err.Error(), "malformed requirement: found only malformed ID: \"\\nREQ-0.TEST-SYS-007")
	assert.Contains(t, err.Error(), "malformed requirement: found only malformed ID: \"\\nREQ-0-TESTSYS-008")

	assert.Contains(t, err.Error(), "Invalid requirement sequence number for REQ-0-TEST-SYS-001, is duplicate.")
	assert.Contains(t, err.Error(), "Invalid requirement sequence number for REQ-0-TEST-SYS-013: missing requirements in between. Total number of requirements is 10.")

	assert.Contains(t, err.Error(), "Requirement REQ-0-TEST-SWH-006 in file /reqtraq/testdata/TestPreCommitCreateReqGraph/0-TEST-211-SRD.lyx has no parents.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-0-TEST-SWH-009: REQ-0-TEST-SYS-003 does not exist.")

	assert.Contains(t, err.Error(), "REQ-0-TEST-SWL-003 is deleted.")
	assert.Contains(t, err.Error(), "REQ-0-TEST-SWL-022 does not exist.")

	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-0-TEST-SWH-004: REQ-0-TEST-SYS-022 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-0-TEST-SWH-008: REQ-0-TEST-SYS-003 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-0-TEST-SWH-002: REQ-0-TEST-SYS-002 is deleted.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-0-TEST-SWL-002: REQ-0-TEST-SYS-002 is deleted.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-0-TEST-SWH-005: REQ-0-TEST-SYS-003 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-0-TEST-SWH-010: REQ-0-TEST-SYS-003 does not exist.")
	assert.Contains(t, err.Error(), "Invalid parent of requirement REQ-0-TEST-SWH-011: REQ-0-TEST-SYS-003 does not exist.")

	assert.Contains(t, err.Error(), "Requirement REQ-0-TEST-SWH-007 in file /reqtraq/testdata/TestPreCommitCreateReqGraph/0-TEST-211-SRD.lyx has no parents.")
}

func TestPreCommitCheckReqReferences(t *testing.T) {
	err := precommit(git.RepoPath()+"/reqtraq/testdata/TestPreCommitCheckReqReferences", git.RepoPath()+"/reqtraq/testdata/TestPreCommitCheckReqReferences", git.RepoPath()+"/certdocs/attributes.json")
	assert.NotNil(t, err, "Invalid json path, should be rejected")

	nLines := strings.Count(err.Error(), "\n")
	assert.Equal(t, 5, nLines, "Number of errors is not correct.")

	assert.Contains(t, err.Error(), "Invalid reference to inexistent requirement REQ-0-TEST-SYS-022")
	assert.Contains(t, err.Error(), "Invalid reference to deleted requirement REQ-0-TEST-SYS-002")
	assert.Contains(t, err.Error(), "Requirement 'REQ-0-TEST-SWH-006' is missing attribute 'Verification'.")
	assert.Contains(t, err.Error(), "Requirement 'REQ-0-TEST-SWH-008' has invalid value 'gibberish.' in attribute 'VERIFICATION'.")
	assert.Contains(t, err.Error(), "Requirement 'REQ-0-TEST-SWH-007' is missing attribute 'Safety Impact'.")
}
