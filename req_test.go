package main

import (
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/stretchr/testify/assert"
)

func TestReqGraph_OrdsByPosition(t *testing.T) {
	rg := ReqGraph{Reqs: make(map[string]*Req)}

	r := &Req{ID: "REQ-TEST-SYS-2", Level: config.SYSTEM, Position: 1}
	rg.Reqs[r.ID] = r

	r = &Req{ID: "REQ-TEST-SYS-1", Level: config.SYSTEM, Position: 2}
	rg.Reqs[r.ID] = r

	r = &Req{ID: "REQ-TEST-SWH-1", Level: config.HIGH, ParentIds: []string{"REQ-TEST-SYS-1"}}
	rg.Reqs[r.ID] = r

	r = &Req{ID: "REQ-UIEM-SYS-1", Level: config.SYSTEM, ParentIds: []string{"REQ-TEST-SYS-1"}}
	rg.Reqs[r.ID] = r

	assert.Empty(t, rg.resolve())

	reqs := rg.OrdsByPosition()
	assert.Len(t, reqs, 2)
	assert.Equal(t, "REQ-TEST-SYS-2", reqs[0].ID)
	assert.Equal(t, "REQ-TEST-SYS-1", reqs[1].ID)
}

func TestReq_Significant(t *testing.T) {
	tests := []struct {
		filter ReqFilter
		empty  bool
	}{
		{ReqFilter{}, true},
		{ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{}}, true},

		{ReqFilter{IDRegexp: regexp.MustCompile("REQ-TEST-SWH-*")}, false},
		{ReqFilter{TitleRegexp: regexp.MustCompile("thrust")}, false},
		{ReqFilter{BodyRegexp: regexp.MustCompile("thrust")}, false},
		{ReqFilter{AnyAttributeRegexp: regexp.MustCompile("Demo*")}, false},
		{ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Demo*")}}, false},
	}

	for _, test := range tests {
		if test.empty {
			assert.True(t, test.filter.IsEmpty(), "filter is not empty: %v", test.filter)
		} else {
			assert.False(t, test.filter.IsEmpty(), "filter is empty: %v", test.filter)
		}
	}
}

func TestParsing(t *testing.T) {
	repoName := repos.RegisterRepository(filepath.Join(repos.BaseRepoPath(), "testdata"))
	document := config.Document{
		Path: "valid_system_requirement/TEST-100-ORD.md",
	}

	// test a valid requirements document
	rg := &ReqGraph{Reqs: make(map[string]*Req)}

	err := rg.addCertdocToGraph(repoName, &document)
	if err != nil {
		t.Errorf("parseCertdocToGraph: %v", err)
	}
	assert.Empty(t, rg.Errors, "Unexpected errors while parsing "+document.Path)

	var systemReqs [15]Req
	for i := 0; i < 15; i++ {
		reqNo := strconv.Itoa(i + 1)
		reqPos := i
		if i > 0 {
			// Assumptions are not returned in the OrdsByPosition list
			reqPos = i + 1
		}
		systemReqs[i] = Req{ID: "REQ-TEST-SYS-" + reqNo,
			Prefix:   "REQ",
			Level:    config.SYSTEM,
			Document: &document,
			RepoName: repoName,
			Position: reqPos,
			Attributes: map[string]string{
				"SAFETY IMPACT": "Impact " + reqNo,
				"RATIONALE":     "Rationale " + reqNo,
				"VERIFICATION":  "Test " + reqNo},
		}
	}

	assert.Equal(t, len(systemReqs), len(rg.Reqs), "Requirement count mismatch")

	for i, systemReq := range rg.OrdsByPosition() {
		if systemReqs[i].ID != systemReq.ID || systemReqs[i].Level != systemReq.Level || systemReqs[i].Document != systemReq.Document || systemReqs[i].Position != systemReq.Position || systemReqs[i].RepoName != systemReq.RepoName {
			t.Errorf("Invalid system requirement\nExpected %#v,\n   got %#v", systemReqs[i], systemReq)
		}
	}

	document = config.Document{
		Path: "invalid_system_requirement/NAM1-100-ORD.md",
	}

	// an invalid requirements document containing requirement naming errors
	rg = &ReqGraph{Reqs: make(map[string]*Req)}

	err = rg.addCertdocToGraph(repoName, &document)
	if err != nil {
		t.Errorf("parseCertdocToGraph: %v", err)
	}
	assert.Equal(t, 3, len(rg.Errors))
	assert.Contains(t, rg.Errors[0].Error(), "Incorrect project abbreviation for requirement REQ-NAN1-SYS-1. Expected NAM1, got NAN1.")
	assert.Contains(t, rg.Errors[1].Error(), "Incorrect requirement type for requirement REQ-NAM1-SWH-2. Expected SYS, got SWH.")
	assert.Contains(t, rg.Errors[2].Error(), "Requirement number cannot begin with a 0: REQ-NAM1-SYS-03. Got 03.")

	// an invalid requirements document containing sequence errors
	rg = &ReqGraph{Reqs: make(map[string]*Req)}

	document = config.Document{
		Path: "invalid_system_requirement/GAP1-100-ORD.md",
	}

	err = rg.addCertdocToGraph(repoName, &document)
	if err != nil {
		t.Errorf("parseCertdocToGraph: %v", err)
	}
	assert.Equal(t, 2, len(rg.Errors))
	assert.Contains(t, rg.Errors[0].Error(), "Invalid requirement sequence number for REQ-GAP1-SYS-3: missing requirements in between. Expected ID Number 2.")
	assert.Contains(t, rg.Errors[1].Error(), "Invalid requirement sequence number for REQ-GAP1-SYS-6: missing requirements in between. Expected ID Number 5.")

	// an invalid requirements document containing duplicates
	rg = &ReqGraph{Reqs: make(map[string]*Req)}

	document = config.Document{
		Path: "invalid_system_requirement/DUP1-100-ORD.md",
	}

	err = rg.addCertdocToGraph(repoName, &document)
	if err != nil {
		t.Errorf("parseCertdocToGraph: %v", err)
	}
	assert.Equal(t, 3, len(rg.Errors))
	assert.Contains(t, rg.Errors[0].Error(), "Invalid requirement sequence number for REQ-DUP1-SYS-1, is duplicate.")
	assert.Contains(t, rg.Errors[1].Error(), "Invalid requirement sequence number for REQ-DUP1-SYS-2, is duplicate.")
	assert.Contains(t, rg.Errors[2].Error(), "Invalid requirement sequence number for REQ-DUP1-SYS-3, is duplicate.")
}

func TestReq_IsDeleted(t *testing.T) {
	req := Req{ID: "REQ-TEST-SYS-2", Title: "DELETED"}
	assert.True(t, req.IsDeleted(), "Requirement with title %s should have status DELETED", req.Title)
	req = Req{ID: "REQ-TEST-SYS-2", Title: "DELETED Requirement"}
	assert.True(t, req.IsDeleted(), "Requirement with title %s should have status DELETED", req.Title)

	req = Req{ID: "REQ-TEST-SYS-2", Title: "Deleted Requirements"}
	assert.False(t, req.IsDeleted(), "Requirement with title %s should NOT have status DELETED", req.Title)
}
