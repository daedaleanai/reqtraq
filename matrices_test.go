package main

import (
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

	// System reqs
	rg.Reqs["REQ-TEST-SYS-1"] = &Req{
		ID:       "REQ-TEST-SYS-1",
		IDNumber: 1,
		Level:    config.SYSTEM,
		Document: &config.Document{},
	}
	rg.Reqs["REQ-TEST-SYS-2"] = &Req{
		ID:       "REQ-TEST-SYS-2",
		IDNumber: 2,
		Level:    config.SYSTEM,
		Document: &config.Document{},
	}

	// High level requirements
	rg.Reqs["REQ-TEST-SWH-1"] = &Req{
		ID:       "REQ-TEST-SWH-1",
		IDNumber: 1,
		Level:    config.HIGH,
		Document: &config.Document{},
	}
	rg.Reqs["REQ-TEST-SWH-2"] = &Req{
		ID:        "REQ-TEST-SWH-2",
		IDNumber:  2,
		Level:     config.HIGH,
		ParentIds: []string{"REQ-TEST-SYS-1"},
		Document:  &config.Document{},
	}
	rg.Reqs["REQ-TEST-SWH-3"] = &Req{
		ID:        "REQ-TEST-SWH-3",
		IDNumber:  3,
		Level:     config.HIGH,
		ParentIds: []string{"REQ-TEST-SYS-1"},
		Document:  &config.Document{},
	}

	// Low level requirements
	rg.Reqs["REQ-TEST-SWL-1"] = &Req{
		ID:        "REQ-TEST-SWL-1",
		IDNumber:  1,
		Level:     config.LOW,
		ParentIds: []string{"REQ-TEST-SWH-2"},
		Document:  &config.Document{},
	}
	rg.Reqs["REQ-TEST-SWL-2"] = &Req{
		ID:       "REQ-TEST-SWL-2",
		IDNumber: 2,
		Level:    config.LOW,
		ParentIds: []string{
			"REQ-TEST-SWH-1",
			"REQ-TEST-SWH-2",
		},
		Document: &config.Document{},
	}
	rg.Reqs["REQ-TEST-SWL-3"] = &Req{
		ID:        "REQ-TEST-SWL-3",
		IDNumber:  3,
		Level:     config.LOW,
		ParentIds: []string{},
		Document:  &config.Document{},
	}

	errs := SortErrs(rg.resolve())
	assert.Equal(t, 0, len(errs))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SYS-1 -> REQ-TEST-SWH-2",
			"REQ-TEST-SYS-1 -> REQ-TEST-SWH-3",
			"REQ-TEST-SYS-2 -> NIL",
		},
		rg.matrixRows(rg.createDownstreamMatrix(config.SYSTEM, config.HIGH)))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SWH-1 -> NIL",
			"REQ-TEST-SWH-2 -> REQ-TEST-SYS-1",
			"REQ-TEST-SWH-3 -> REQ-TEST-SYS-1",
		},
		rg.matrixRows(rg.createUpstreamMatrix(config.HIGH, config.SYSTEM)))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SWH-1 -> REQ-TEST-SWL-2",
			"REQ-TEST-SWH-2 -> REQ-TEST-SWL-1",
			"REQ-TEST-SWH-2 -> REQ-TEST-SWL-2",
			"REQ-TEST-SWH-3 -> NIL",
		},
		rg.matrixRows(rg.createDownstreamMatrix(config.HIGH, config.LOW)))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SWL-1 -> REQ-TEST-SWH-2",
			"REQ-TEST-SWL-2 -> REQ-TEST-SWH-1",
			"REQ-TEST-SWL-2 -> REQ-TEST-SWH-2",
			"REQ-TEST-SWL-3 -> NIL",
		},
		rg.matrixRows(rg.createUpstreamMatrix(config.LOW, config.HIGH)))
}
