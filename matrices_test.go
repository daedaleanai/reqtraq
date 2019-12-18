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
func (rg *reqGraph) matrixRows(matrix [][2]*MatrixItem) []string {
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
	rg := &reqGraph{Reqs: make(map[string]*Req)}
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SYS-2", IDNumber: 2, Level: config.SYSTEM}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SYS-1", IDNumber: 1, Level: config.SYSTEM}, "./TRAQ-0-SRD.md"))

	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWH-2", IDNumber: 2, Level: config.HIGH, ParentIds: []string{"REQ-TRAQ-SYS-1"}}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWH-1", IDNumber: 1, Level: config.HIGH}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWH-3", IDNumber: 3, Level: config.HIGH, ParentIds: []string{"REQ-TRAQ-SYS-1"}}, "./TRAQ-0-SRD.md"))

	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWL-3", IDNumber: 3, Level: config.LOW}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWL-1", IDNumber: 1, Level: config.LOW, ParentIds: []string{"REQ-TRAQ-SWH-2"}}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWL-2", IDNumber: 2, Level: config.LOW, ParentIds: []string{"REQ-TRAQ-SWH-1", "REQ-TRAQ-SWH-2"}}, "./TRAQ-0-SRD.md"))

	errs := SortErrs(rg.Resolve())
	assert.Equal(t, 2, len(errs))
	assert.Equal(t, "Requirement REQ-TRAQ-SWH-1 in file ./TRAQ-0-SRD.md has no parents.", errs[0])
	assert.Equal(t, "Requirement REQ-TRAQ-SWL-3 in file ./TRAQ-0-SRD.md has no parents.", errs[1])

	assert.Equal(t,
		[]string{
			"REQ-TRAQ-SYS-1 -> REQ-TRAQ-SWH-2",
			"REQ-TRAQ-SYS-1 -> REQ-TRAQ-SWH-3",
			"REQ-TRAQ-SYS-2 -> NIL",
		},
		rg.matrixRows(rg.createDownstreamMatrix(config.SYSTEM, config.HIGH)))

	assert.Equal(t,
		[]string{
			"REQ-TRAQ-SWH-1 -> NIL",
			"REQ-TRAQ-SWH-2 -> REQ-TRAQ-SYS-1",
			"REQ-TRAQ-SWH-3 -> REQ-TRAQ-SYS-1",
		},
		rg.matrixRows(rg.createUpstreamMatrix(config.HIGH, config.SYSTEM)))

	assert.Equal(t,
		[]string{
			"REQ-TRAQ-SWH-1 -> REQ-TRAQ-SWL-2",
			"REQ-TRAQ-SWH-2 -> REQ-TRAQ-SWL-1",
			"REQ-TRAQ-SWH-2 -> REQ-TRAQ-SWL-2",
			"REQ-TRAQ-SWH-3 -> NIL",
		},
		rg.matrixRows(rg.createDownstreamMatrix(config.HIGH, config.LOW)))

	assert.Equal(t,
		[]string{
			"REQ-TRAQ-SWL-1 -> REQ-TRAQ-SWH-2",
			"REQ-TRAQ-SWL-2 -> REQ-TRAQ-SWH-1",
			"REQ-TRAQ-SWL-2 -> REQ-TRAQ-SWH-2",
			"REQ-TRAQ-SWL-3 -> NIL",
		},
		rg.matrixRows(rg.createUpstreamMatrix(config.LOW, config.HIGH)))
}
