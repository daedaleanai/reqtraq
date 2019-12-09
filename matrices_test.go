package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/stretchr/testify/assert"
)

func matrixIDs(matrix [][2]*Req) []string {
	parts := make([]string, 0)
	for _, reqs := range matrix {
		e := make([]string, 0)
		for _, r := range reqs {
			if r == nil {
				e = append(e, "NIL")
			} else {
				e = append(e, r.ID)
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
	rg := reqGraph{Reqs: make(map[string]*Req)}
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SYS-2", Level: config.SYSTEM}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SYS-1", Level: config.SYSTEM}, "./TRAQ-0-SRD.md"))

	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWH-2", Level: config.HIGH, ParentIds: []string{"REQ-TRAQ-SYS-1"}}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWH-1", Level: config.HIGH}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWH-3", Level: config.HIGH, ParentIds: []string{"REQ-TRAQ-SYS-1"}}, "./TRAQ-0-SRD.md"))

	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWL-3", Level: config.LOW}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWL-1", Level: config.LOW, ParentIds: []string{"REQ-TRAQ-SWH-2"}}, "./TRAQ-0-SRD.md"))
	assert.NoError(t, rg.AddReq(&Req{ID: "REQ-TRAQ-SWL-2", Level: config.LOW, ParentIds: []string{"REQ-TRAQ-SWH-1", "REQ-TRAQ-SWH-2"}}, "./TRAQ-0-SRD.md"))

	errs := SortErrs(rg.Resolve())
	assert.Equal(t, 2, len(errs))
	assert.Equal(t, "Requirement REQ-TRAQ-SWH-1 in file ./TRAQ-0-SRD.md has no parents.", errs[0])
	assert.Equal(t, "Requirement REQ-TRAQ-SWL-3 in file ./TRAQ-0-SRD.md has no parents.", errs[1])

	sys1 := rg.Reqs["REQ-TRAQ-SYS-1"]
	sys2 := rg.Reqs["REQ-TRAQ-SYS-2"]
	swh1 := rg.Reqs["REQ-TRAQ-SWH-1"]
	swh2 := rg.Reqs["REQ-TRAQ-SWH-2"]
	swh3 := rg.Reqs["REQ-TRAQ-SWH-3"]
	swl1 := rg.Reqs["REQ-TRAQ-SWL-1"]
	swl2 := rg.Reqs["REQ-TRAQ-SWL-2"]
	swl3 := rg.Reqs["REQ-TRAQ-SWL-3"]

	assert.Equal(t,
		matrixIDs([][2]*Req{
			{sys1, swh2},
			{sys1, swh3},
			{sys2, nil},
		}),
		matrixIDs(rg.createMatrix(config.SYSTEM, config.HIGH)))

	assert.Equal(t,
		matrixIDs([][2]*Req{
			{swh1, nil},
			{swh2, sys1},
			{swh3, sys1},
		}),
		matrixIDs(rg.createMatrix(config.HIGH, config.SYSTEM)))

	assert.Equal(t,
		matrixIDs([][2]*Req{
			{swh1, swl2},
			{swh2, swl1},
			{swh2, swl2},
			{swh3, nil},
		}),
		matrixIDs(rg.createMatrix(config.HIGH, config.LOW)))

	assert.Equal(t,
		matrixIDs([][2]*Req{
			{swl1, swh2},
			{swl2, swh1},
			{swl2, swh2},
			{swl3, nil},
		}),
		matrixIDs(rg.createMatrix(config.LOW, config.HIGH)))
}
