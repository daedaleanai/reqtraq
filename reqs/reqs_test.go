package reqs

import (
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/stretchr/testify/assert"
)

// @llr REQ-TRAQ-SWL-45, REQ-TRAQ-SWL-77
func TestReqGraph_OrdsByPosition(t *testing.T) {
	rg := ReqGraph{Reqs: make(map[string]*Req)}

	sysDoc := config.Document{
		Path: "path/to/sys.md",
		ReqSpec: config.ReqSpec{
			Prefix: "TEST",
			Level:  "SYS",
		},
		Schema: config.Schema{
			Requirements: regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
			Attributes:   make(map[string]*config.Attribute),
		},
	}

	r := &Req{ID: "REQ-TEST-SYS-2", Position: 1, Document: &sysDoc, Body: "Shall"}
	rg.Reqs[r.ID] = r

	r = &Req{ID: "REQ-TEST-SYS-1", Position: 2, Document: &sysDoc, Body: "Shall shall"}
	rg.Reqs[r.ID] = r

	srdDoc := config.Document{
		Path: "path/to/srd.md",
		ReqSpec: config.ReqSpec{
			Prefix: "TEST",
			Level:  "SWH",
		},
		LinkSpecs: []config.LinkSpec{
			{
				Child: config.ReqSpec{
					Prefix:  config.ReqPrefix("TEST"),
					Level:   config.ReqLevel("SWH"),
					Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
					AttrKey: "",
					AttrVal: regexp.MustCompile(".*")},
				Parent: config.ReqSpec{
					Prefix:  config.ReqPrefix("TEST"),
					Level:   config.ReqLevel("SYS"),
					Re:      regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
					AttrKey: "",
					AttrVal: regexp.MustCompile(".*")},
			},
		},
		Schema: config.Schema{
			Requirements: regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
			Attributes:   make(map[string]*config.Attribute),
		},
	}

	r = &Req{ID: "REQ-TEST-SWH-1", ParentIds: []string{"REQ-TEST-SYS-1"}, Document: &srdDoc, Body: "SHALL"}
	rg.Reqs[r.ID] = r

	r = &Req{ID: "REQ-UIEM-SYS-1", ParentIds: []string{"REQ-TEST-SYS-1"}, Document: &srdDoc, Body: ""}
	rg.Reqs[r.ID] = r

	reqIssues := rg.Resolve()
	assert.Equal(t, len(reqIssues), 4)

	assertIssueExists := func(message string) {
		found := false
		for _, error := range reqIssues {
			if error.Description == message {
				found = true
				break
			}
		}
		assert.True(t, found, "Did not find issue: ", message)
	}

	assertIssueExists("Requirement `REQ-TEST-SYS-1` in document `path/to/sys.md` contains multiple SHALL statements in its body")
	assertIssueExists("Requirement `REQ-UIEM-SYS-1` in document `path/to/srd.md` does not match required regexp `REQ-TEST-SWH-(\\d+)`")
	assertIssueExists("Requirement `REQ-UIEM-SYS-1` in document `path/to/srd.md` does not contain a SHALL statement in its body")
	assertIssueExists("Requirement 'REQ-UIEM-SYS-1' has invalid parent link ID 'REQ-TEST-SYS-1'.")

	reqs := rg.OrdsByPosition()
	assert.Len(t, reqs, 2)
	assert.Equal(t, "REQ-TEST-SYS-2", reqs[0].ID)
	assert.Equal(t, "REQ-TEST-SYS-1", reqs[1].ID)
}

// @llr REQ-TRAQ-SWL-20, REQ-TRAQ-SWL-21
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

// @llr REQ-TRAQ-SWL-27
func TestParsing(t *testing.T) {
	repoPath := repos.RepoPath(filepath.Join(string(repos.BaseRepoPath()), "testdata"))
	repoName := repos.RepoName("testdata")
	repos.RegisterRepository(repoName, repoPath)
	document := config.Document{
		Path: "valid_system_requirement/TEST-100-ORD.md",
		ReqSpec: config.ReqSpec{
			Prefix: "TEST",
			Level:  "SYS",
		},
	}

	// test a valid requirements document
	rg := &ReqGraph{Reqs: make(map[string]*Req)}

	err := rg.addCertdocToGraph(repoName, &document)
	if err != nil {
		t.Errorf("parseCertdocToGraph: %v", err)
	}
	assert.Empty(t, rg.Issues, "Unexpected errors while parsing "+document.Path)

	// The last  index is -1 because OrdsByPosition does not return assumptions. If it did,
	// it should not match
	requirementPositions := []int{7, 16, 32, 41, 51, 65, 66, 67, 68, 73, 82, 96, 97, 98, -1}
	var systemReqs [15]Req
	for i := 0; i < 15; i++ {
		reqNo := strconv.Itoa(i + 1)
		systemReqs[i] = Req{ID: "REQ-TEST-SYS-" + reqNo,
			Variant:  ReqVariantRequirement,
			Document: &document,
			RepoName: repoName,
			Position: requirementPositions[i],
			Attributes: map[string]string{
				"SAFETY IMPACT": "Impact " + reqNo,
				"RATIONALE":     "Rationale " + reqNo,
				"VERIFICATION":  "Test " + reqNo},
		}
	}

	assert.Equal(t, len(systemReqs), len(rg.Reqs), "Requirement count mismatch")

	for i, systemReq := range rg.OrdsByPosition() {
		if systemReqs[i].ID != systemReq.ID || systemReqs[i].Document != systemReq.Document || systemReqs[i].Position != systemReq.Position || systemReqs[i].RepoName != systemReq.RepoName {
			t.Errorf("Invalid system requirement\nExpected %#v,\n   got %#v", systemReqs[i], systemReq)
		}
	}

	document = config.Document{
		Path: "invalid_system_requirement/NAM1-100-ORD.md",
		ReqSpec: config.ReqSpec{
			Prefix: "NAM1",
			Level:  "SYS",
		},
	}

	// an invalid requirements document containing requirement naming errors
	rg = &ReqGraph{Reqs: make(map[string]*Req)}

	err = rg.addCertdocToGraph(repoName, &document)
	if err != nil {
		t.Errorf("parseCertdocToGraph: %v", err)
	}
	assert.Equal(t, 3, len(rg.Issues))

	assertIssueExists := func(message string) {
		found := false
		for _, error := range rg.Issues {
			if error.Description == message {
				found = true
				break
			}
		}
		assert.True(t, found, "Did not find issue: ", message)
	}
	assertIssueExists("Incorrect project abbreviation for requirement REQ-NAN1-SYS-1. Expected NAM1, got NAN1.")
	assertIssueExists("Incorrect requirement type for requirement REQ-NAM1-SWH-2. Expected SYS, got SWH.")
	assertIssueExists("Requirement number cannot begin with a 0: REQ-NAM1-SYS-03. Got 03.")

	// an invalid requirements document containing sequence errors
	rg = &ReqGraph{Reqs: make(map[string]*Req)}

	document = config.Document{
		Path: "invalid_system_requirement/GAP1-100-ORD.md",
		ReqSpec: config.ReqSpec{
			Prefix: "GAP1",
			Level:  "SYS",
		},
	}

	err = rg.addCertdocToGraph(repoName, &document)
	if err != nil {
		t.Errorf("parseCertdocToGraph: %v", err)
	}
	assert.Equal(t, 2, len(rg.Issues))
	assertIssueExists("Invalid requirement sequence number for REQ-GAP1-SYS-3: missing requirements in between. Expected ID Number 2.")
	assertIssueExists("Invalid requirement sequence number for REQ-GAP1-SYS-6: missing requirements in between. Expected ID Number 5.")

	// an invalid requirements document containing duplicates
	rg = &ReqGraph{Reqs: make(map[string]*Req)}

	document = config.Document{
		Path: "invalid_system_requirement/DUP1-100-ORD.md",
		ReqSpec: config.ReqSpec{
			Prefix: "DUP1",
			Level:  "SYS",
		},
	}

	err = rg.addCertdocToGraph(repoName, &document)
	if err != nil {
		t.Errorf("parseCertdocToGraph: %v", err)
	}
	assert.Equal(t, 3, len(rg.Issues))
	assertIssueExists("Invalid requirement sequence number for REQ-DUP1-SYS-1, is duplicate.")
	assertIssueExists("Invalid requirement sequence number for REQ-DUP1-SYS-2, is duplicate.")
	assertIssueExists("Invalid requirement sequence number for REQ-DUP1-SYS-3, is duplicate.")
}

// @llr REQ-TRAQ-SWL-23
func TestReq_IsDeleted(t *testing.T) {
	req := Req{ID: "REQ-TEST-SYS-2", Title: "DELETED"}
	assert.True(t, req.IsDeleted(), "Requirement with title %s should have status DELETED", req.Title)
	req = Req{ID: "REQ-TEST-SYS-2", Title: "DELETED Requirement"}
	assert.True(t, req.IsDeleted(), "Requirement with title %s should have status DELETED", req.Title)

	req = Req{ID: "REQ-TEST-SYS-2", Title: "Deleted Requirements"}
	assert.False(t, req.IsDeleted(), "Requirement with title %s should NOT have status DELETED", req.Title)
}
