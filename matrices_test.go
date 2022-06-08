package main

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/stretchr/testify/assert"
)

// matrixRows creates a simple textual representation of the matrix,
// for comparison purposes.
func (rg *ReqGraph) matrixRows(matrix []TableRow) []string {
	rg.sortMatrices(matrix)
	parts := make([]string, 0)
	for _, reqs := range matrix {
		e := make([]string, 0)
		for _, r := range reqs {
			if r == nil {
				e = append(e, "NIL")
			} else {
				e = append(e, r.Name)
			}
		}
		parts = append(parts, strings.Join(e, " -> "))
	}
	return parts
}

func SortErrs(errs []error) []string {
	res := make([]string, len(errs))
	for i, err := range errs {
		res[i] = err.Error()
	}
	sort.Strings(res)
	return res
}

func TestReqGraph_createMatrix(t *testing.T) {
	rg := &ReqGraph{Reqs: make(map[string]*Req)}

	sysReqSpec := config.ReqSpec{
		Prefix: "SYS",
		Level:  "TEST",
	}
	swhReqSpec := config.ReqSpec{
		Prefix: "SWH",
		Level:  "TEST",
	}
	swlReqSpec := config.ReqSpec{
		Prefix: "SWL",
		Level:  "TEST",
	}

	sysDoc := config.Document{
		Path:    "path/to/sys.md",
		ReqSpec: sysReqSpec,
		Schema: config.Schema{
			Requirements: regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
			Attributes:   make(map[string]*config.Attribute),
		},
	}

	// System reqs
	rg.Reqs["REQ-TEST-SYS-1"] = &Req{
		ID:       "REQ-TEST-SYS-1",
		IDNumber: 1,
		Document: &sysDoc,
	}
	rg.Reqs["REQ-TEST-SYS-2"] = &Req{
		ID:       "REQ-TEST-SYS-2",
		IDNumber: 2,
		Document: &sysDoc,
	}

	srdDoc := config.Document{
		Path:          "path/to/srd.md",
		ParentReqSpec: sysReqSpec,
		ReqSpec:       swhReqSpec,
		Schema: config.Schema{
			Requirements: regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
			Attributes:   make(map[string]*config.Attribute),
		},
	}

	// High level requirements
	rg.Reqs["REQ-TEST-SWH-1"] = &Req{
		ID:       "REQ-TEST-SWH-1",
		IDNumber: 1,
		Document: &srdDoc,
	}
	rg.Reqs["REQ-TEST-SWH-2"] = &Req{
		ID:        "REQ-TEST-SWH-2",
		IDNumber:  2,
		ParentIds: []string{"REQ-TEST-SYS-1"},
		Document:  &srdDoc,
	}
	rg.Reqs["REQ-TEST-SWH-3"] = &Req{
		ID:        "REQ-TEST-SWH-3",
		IDNumber:  3,
		ParentIds: []string{"REQ-TEST-SYS-1"},
		Document:  &srdDoc,
	}

	sddDoc := config.Document{
		Path:          "path/to/sdd.md",
		ParentReqSpec: swhReqSpec,
		ReqSpec:       swlReqSpec,
		Schema: config.Schema{
			Requirements: regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
			Attributes:   make(map[string]*config.Attribute),
		},
	}

	// Low level requirements
	rg.Reqs["REQ-TEST-SWL-1"] = &Req{
		ID:        "REQ-TEST-SWL-1",
		IDNumber:  1,
		ParentIds: []string{"REQ-TEST-SWH-2"},
		Document:  &sddDoc,
	}
	rg.Reqs["REQ-TEST-SWL-2"] = &Req{
		ID:       "REQ-TEST-SWL-2",
		IDNumber: 2,
		ParentIds: []string{
			"REQ-TEST-SWH-1",
			"REQ-TEST-SWH-2",
		},
		Document: &sddDoc,
	}
	rg.Reqs["REQ-TEST-SWL-3"] = &Req{
		ID:        "REQ-TEST-SWL-3",
		IDNumber:  3,
		ParentIds: []string{},
		Document:  &sddDoc,
	}

	errs := SortErrs(rg.resolve())
	assert.Equal(t, 0, len(errs))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SYS-1 -> REQ-TEST-SWH-2",
			"REQ-TEST-SYS-1 -> REQ-TEST-SWH-3",
			"REQ-TEST-SYS-2 -> NIL",
		},
		rg.matrixRows(rg.createDownstreamMatrix(sysReqSpec, swhReqSpec)))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SWH-1 -> NIL",
			"REQ-TEST-SWH-2 -> REQ-TEST-SYS-1",
			"REQ-TEST-SWH-3 -> REQ-TEST-SYS-1",
		},
		rg.matrixRows(rg.createUpstreamMatrix(swhReqSpec, sysReqSpec)))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SWH-1 -> REQ-TEST-SWL-2",
			"REQ-TEST-SWH-2 -> REQ-TEST-SWL-1",
			"REQ-TEST-SWH-2 -> REQ-TEST-SWL-2",
			"REQ-TEST-SWH-3 -> NIL",
		},
		rg.matrixRows(rg.createDownstreamMatrix(swhReqSpec, swlReqSpec)))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SWL-1 -> REQ-TEST-SWH-2",
			"REQ-TEST-SWL-2 -> REQ-TEST-SWH-1",
			"REQ-TEST-SWL-2 -> REQ-TEST-SWH-2",
			"REQ-TEST-SWL-3 -> NIL",
		},
		rg.matrixRows(rg.createUpstreamMatrix(swlReqSpec, swhReqSpec)))
}
