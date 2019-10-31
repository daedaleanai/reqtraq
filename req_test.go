// @tests @llr REQ-TRAQ-SWL-15
package main

import (
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/stretchr/testify/assert"
)

func TestReqGraph_AddReq(t *testing.T) {
	rg := reqGraph{Reqs: make(map[string]*Req)}

	req := &Req{ID: "REQ-TRAQ-SWH-1"}
	req2 := &Req{ID: "REQ-TRAQ-SWL-1", ParentIds: []string{"REQ-TRAQ-SWH-1"}}

	rg.AddReq(req, "./TRAQ-0-SRD.md")
	rg.AddReq(req2, "./TRAQ-1-SDD.md")

	// if this becomes more complex we can move it into a table of tescases.
	if expectedReq := (&Req{
		ID:   "REQ-TRAQ-SWH-1",
		Path: "./TRAQ-0-SRD.md",
	}); !reflect.DeepEqual(expectedReq, rg.Reqs["REQ-TRAQ-SWH-1"]) {
		t.Errorf("\nexpected %#v,\n     got %#v", expectedReq, rg.Reqs["REQ-TRAQ-SWH-1"])
	}

	if expectedReq := (&Req{
		ID:        "REQ-TRAQ-SWL-1",
		Path:      "./TRAQ-1-SDD.md",
		ParentIds: []string{"REQ-TRAQ-SWH-1"},
	}); !reflect.DeepEqual(expectedReq, rg.Reqs["REQ-TRAQ-SWL-1"]) {
		t.Errorf("\nexpected %#v,\n     got %#v", expectedReq, rg.Reqs["REQ-TRAQ-SWL-1"])
	}
}

func TestReqGraph_AddReq_someMore(t *testing.T) {
	rg := reqGraph{Reqs: make(map[string]*Req)}

	for _, v := range []*Req{
		{ID: "REQ-TRAQ-SWH-1", Position: 1},
		{ID: "REQ-TRAQ-SWH-2", Position: 2},
		{ID: "REQ-TRAQ-SWH-3", Position: 3},
	} {
		if err := rg.AddReq(v, "./TRAQ-0-SRD.md"); err != nil {
			t.Errorf("addreq: %v", err)
		}
	}

	for _, v := range []*Req{
		{ID: "REQ-TRAQ-SWL-1", ParentIds: []string{"REQ-TRAQ-SWH-1"}, Position: 1},
		{ID: "REQ-TRAQ-SWL-2", ParentIds: []string{"REQ-TRAQ-SWH-1"}, Position: 2},
		{ID: "REQ-TRAQ-SWL-3", ParentIds: []string{"REQ-TRAQ-SWH-3"}, Position: 3},
	} {
		if err := rg.AddReq(v, "./TRAQ-1-SDD.md"); err != nil {
			t.Errorf("addreq: %v", err)
		}
	}

	for i, v := range []struct {
		id     string
		expect Req
	}{
		{"REQ-TRAQ-SWH-1", Req{ID: "REQ-TRAQ-SWH-1", Path: "./TRAQ-0-SRD.md", Position: 1}},
		{"REQ-TRAQ-SWL-1", Req{
			ID:        "REQ-TRAQ-SWL-1",
			Path:      "./TRAQ-1-SDD.md",
			ParentIds: []string{"REQ-TRAQ-SWH-1"},
			Position:  1,
		}},
	} {
		if !reflect.DeepEqual(v.expect, *rg.Reqs[v.id]) {
			t.Errorf("case %d:\nexpected %#v,\n     got %#v", i, v.expect, *rg.Reqs[v.id])
		}
	}
}

func TestReqGraph_OrdsByPosition(t *testing.T) {
	rg := reqGraph{Reqs: make(map[string]*Req)}
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SYS-2", Level: config.SYSTEM, Position: 1}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SYS-1", Level: config.SYSTEM, Position: 2}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWH-1", Level: config.HIGH, ParentIds: []string{"REQ-TRAQ-SYS-1"}}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-UIEM-SYS-1", Level: config.SYSTEM, ParentIds: []string{"REQ-TRAQ-SYS-1"}}, "./TRAQ-0-SRD.md"))
	assert.Empty(t, rg.Resolve())

	reqs := rg.OrdsByPosition()
	assert.Len(t, reqs, 2)
	assert.Equal(t, "REQ-TRAQ-SYS-2", reqs[0].ID)
	assert.Equal(t, "REQ-TRAQ-SYS-1", reqs[1].ID)
}

func TestReq_ReqType(t *testing.T) {
	tests := []struct {
		req     Req
		reqType string
	}{
		{Req{ID: "REQ-TRAQ-SWL-1"}, "SWL"},
		{Req{ID: "Garbage"}, ""},
	}

	for _, test := range tests {
		assert.Equal(t, test.reqType, test.req.ReqType())
	}
}

func TestReq_Significant(t *testing.T) {
	tests := []struct {
		filter ReqFilter
		empty  bool
	}{
		{ReqFilter{}, true},
		{ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{}}, true},

		{ReqFilter{IDRegexp: regexp.MustCompile("REQ-TRAQ-SWH-*")}, false},
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

func TestReq_Matches_filter(t *testing.T) {
	tests := []struct {
		req     Req
		filter  ReqFilter
		diffs   map[string][]string
		matches bool
	}{
		{Req{ID: "REQ-TRAQ-SWH-1", Body: "thrust control"},
			ReqFilter{IDRegexp: regexp.MustCompile("REQ-TRAQ-SWH-*")},
			nil,
			true},
		{Req{ID: "REQ-TRAQ-SWH-1", Title: "The control unit will calculate thrust.", Body: "It will also do much more."},
			ReqFilter{TitleRegexp: regexp.MustCompile("thrust")},
			nil,
			true},
		{Req{ID: "REQ-TRAQ-SWH-1", Title: "The control unit will calculate vertical take off speed.", Body: "It will also output thrust."},
			ReqFilter{TitleRegexp: regexp.MustCompile("thrust")},
			nil,
			false},
		{Req{ID: "REQ-TRAQ-SWH-1", Body: "thrust control"},
			ReqFilter{BodyRegexp: regexp.MustCompile("thrust")},
			nil,
			true},
		{Req{ID: "REQ-TRAQ-SWL-14", Body: "thrust control"},
			ReqFilter{IDRegexp: regexp.MustCompile("REQ-*"), BodyRegexp: regexp.MustCompile("thrust")},
			nil,
			true},
		{Req{ID: "REQ-TRAQ-SWL-14", Body: "thrust control"},
			ReqFilter{IDRegexp: regexp.MustCompile("REQ-DDLN-*"), BodyRegexp: regexp.MustCompile("thrust")},
			nil,
			false},

		// filter attributes
		{Req{ID: "REQ-TRAQ-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			ReqFilter{AnyAttributeRegexp: regexp.MustCompile("Demo*")},
			nil,
			true},
		{Req{ID: "REQ-TRAQ-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			ReqFilter{AnyAttributeRegexp: regexp.MustCompile("Test*")},
			nil,
			false},
		{Req{ID: "REQ-TRAQ-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Demo*")}},
			nil,
			true},
		{Req{ID: "REQ-TRAQ-SWL-14", Attributes: map[string]string{"Color": "Brown"}},
			ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Demo*")}},
			nil,
			false},
		{Req{ID: "REQ-TRAQ-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Test*")}},
			nil,
			false},

		// diffs
		{Req{ID: "REQ-TRAQ-SWL-14", Body: "thrust control"},
			ReqFilter{},
			map[string][]string{"REQ-TRAQ-SWL-1": make([]string, 0)},
			false},
		{Req{ID: "REQ-TRAQ-SWL-14", Body: "thrust control"},
			ReqFilter{},
			map[string][]string{"REQ-TRAQ-SWL-14": make([]string, 0)},
			true},
		{Req{ID: "REQ-TRAQ-SWL-14", Body: "thrust control"},
			ReqFilter{IDRegexp: regexp.MustCompile("X")},
			map[string][]string{"REQ-TRAQ-SWL-14": make([]string, 0)},
			false},
	}

	for _, test := range tests {
		if test.matches {
			assert.True(t, test.req.Matches(&test.filter, test.diffs), "expected requirement to match: %v filter=%v diffs=%v", test.req, test.filter, test.diffs)
		} else {
			assert.False(t, test.req.Matches(&test.filter, test.diffs), "expected requirement to not match: %v filter=%v diffs=%v", test.req, test.filter, test.diffs)
		}
	}
}

// @tests @llr REQ-TRAQ-SWL-15
func TestParsing(t *testing.T) {
	f := "testdata/valid_system_requirement/TEST-100-ORD.md"
	rg := &reqGraph{Reqs: make(map[string]*Req)}
	errors, _ := parseCertdocToGraph(f, rg)
	assert.Empty(t, errors, "Unexpected errors while parsing "+f)
	var systemReqs [5]Req
	for i := 0; i < 5; i++ {
		reqNo := strconv.Itoa(i + 1)
		systemReqs[i] = Req{ID: "REQ-TEST-SYS-" + reqNo,
			Level:    config.SYSTEM,
			Path:     f,
			Position: i,
			Attributes: map[string]string{
				"SAFETY IMPACT": "Impact " + reqNo,
				"RATIONALE":     "Rationale " + reqNo,
				"VERIFICATION":  "Test " + reqNo},
		}
	}

	for i, systemReq := range rg.OrdsByPosition() {
		if systemReqs[i].ID != systemReq.ID || systemReqs[i].Level != systemReq.Level || systemReqs[i].Path != systemReq.Path || systemReqs[i].Position != systemReq.Position {
			t.Errorf("Invalid system requirement\nExpected %#v,\n   got %#v", systemReqs[i], systemReq)
		}
	}
}

// @llr REQ-TRAQ-SWL-17
func TestReq_IsDeleted(t *testing.T) {
	req := Req{ID: "REQ-TEST-SYS-2", Title: "DELETED"}
	assert.True(t, req.IsDeleted(), "Requirement with title %s should have status DELETED", req.Title)
	req = Req{ID: "REQ-TEST-SYS-2", Title: "DELETED Requirement"}
	assert.True(t, req.IsDeleted(), "Requirement with title %s should have status DELETED", req.Title)

	req = Req{ID: "REQ-TEST-SYS-2", Title: "Deleted Requirements"}
	assert.False(t, req.IsDeleted(), "Requirement with title %s should NOT have status DELETED", req.Title)
}
