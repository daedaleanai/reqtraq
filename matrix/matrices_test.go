package matrix

import (
	"regexp"
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/reqs"
	"github.com/stretchr/testify/assert"
)

// matrixRows creates a simple textual representation of the matrix,
// for comparison purposes.
// @llr REQ-TRAQ-SWL-14, REQ-TRAQ-SWL-42, REQ-TRAQ-SWL-43
func matrixRows(rg *reqs.ReqGraph, matrix []TableRow) []string {
	sortMatrices(rg, matrix)
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

// @llr REQ-TRAQ-SWL-14, REQ-TRAQ-SWL-42, REQ-TRAQ-SWL-43
func TestMatrix_createMatrix(t *testing.T) {
	rg := &reqs.ReqGraph{Reqs: make(map[string]*reqs.Req)}

	sysReqSpec := config.ReqSpec{
		Prefix:  "SYS",
		Level:   "TEST",
		Re:      regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
		AttrKey: "",
		AttrVal: regexp.MustCompile(".*"),
	}
	swhReqSpec := config.ReqSpec{
		Prefix:  "SWH",
		Level:   "TEST",
		Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
		AttrKey: "",
		AttrVal: regexp.MustCompile(".*"),
	}
	swlReqSpec := config.ReqSpec{
		Prefix:  "SWL",
		Level:   "TEST",
		Re:      regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
		AttrKey: "",
		AttrVal: regexp.MustCompile(".*"),
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
	rg.Reqs["REQ-TEST-SYS-1"] = &reqs.Req{
		ID:       "REQ-TEST-SYS-1",
		IDNumber: 1,
		Document: &sysDoc,
	}
	rg.Reqs["REQ-TEST-SYS-2"] = &reqs.Req{
		ID:       "REQ-TEST-SYS-2",
		IDNumber: 2,
		Document: &sysDoc,
	}

	srdDoc := config.Document{
		Path:    "path/to/srd.md",
		ReqSpec: swhReqSpec,
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

	// High level requirements
	rg.Reqs["REQ-TEST-SWH-1"] = &reqs.Req{
		ID:       "REQ-TEST-SWH-1",
		IDNumber: 1,
		Document: &srdDoc,
	}
	rg.Reqs["REQ-TEST-SWH-2"] = &reqs.Req{
		ID:        "REQ-TEST-SWH-2",
		IDNumber:  2,
		ParentIds: []string{"REQ-TEST-SYS-1"},
		Document:  &srdDoc,
	}
	rg.Reqs["REQ-TEST-SWH-3"] = &reqs.Req{
		ID:        "REQ-TEST-SWH-3",
		IDNumber:  3,
		ParentIds: []string{"REQ-TEST-SYS-1"},
		Document:  &srdDoc,
	}

	sddDoc := config.Document{
		Path:    "path/to/sdd.md",
		ReqSpec: swlReqSpec,
		LinkSpecs: []config.LinkSpec{
			{
				Child: config.ReqSpec{
					Prefix:  config.ReqPrefix("TEST"),
					Level:   config.ReqLevel("SWL"),
					Re:      regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
					AttrKey: "",
					AttrVal: regexp.MustCompile(".*")},
				Parent: config.ReqSpec{
					Prefix:  config.ReqPrefix("TEST"),
					Level:   config.ReqLevel("SWH"),
					Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
					AttrKey: "",
					AttrVal: regexp.MustCompile(".*")},
			},
		},
		Schema: config.Schema{
			Requirements: regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
			Attributes:   make(map[string]*config.Attribute),
		},
	}

	// Low level requirements
	rg.Reqs["REQ-TEST-SWL-1"] = &reqs.Req{
		ID:        "REQ-TEST-SWL-1",
		IDNumber:  1,
		ParentIds: []string{"REQ-TEST-SWH-2"},
		Document:  &sddDoc,
	}
	rg.Reqs["REQ-TEST-SWL-2"] = &reqs.Req{
		ID:       "REQ-TEST-SWL-2",
		IDNumber: 2,
		ParentIds: []string{
			"REQ-TEST-SWH-1",
			"REQ-TEST-SWH-2",
		},
		Document: &sddDoc,
	}
	rg.Reqs["REQ-TEST-SWL-3"] = &reqs.Req{
		ID:        "REQ-TEST-SWL-3",
		IDNumber:  3,
		ParentIds: []string{},
		Document:  &sddDoc,
	}

	errs := rg.Resolve()
	assert.Equal(t, 0, len(errs))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SYS-1 -> REQ-TEST-SWH-2",
			"REQ-TEST-SYS-1 -> REQ-TEST-SWH-3",
			"REQ-TEST-SYS-2 -> NIL",
		},
		matrixRows(rg, createDownstreamMatrix(rg, sysReqSpec, swhReqSpec)))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SWH-1 -> NIL",
			"REQ-TEST-SWH-2 -> REQ-TEST-SYS-1",
			"REQ-TEST-SWH-3 -> REQ-TEST-SYS-1",
		},
		matrixRows(rg, createUpstreamMatrix(rg, swhReqSpec, sysReqSpec)))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SWH-1 -> REQ-TEST-SWL-2",
			"REQ-TEST-SWH-2 -> REQ-TEST-SWL-1",
			"REQ-TEST-SWH-2 -> REQ-TEST-SWL-2",
			"REQ-TEST-SWH-3 -> NIL",
		},
		matrixRows(rg, createDownstreamMatrix(rg, swhReqSpec, swlReqSpec)))

	assert.Equal(t,
		[]string{
			"REQ-TEST-SWL-1 -> REQ-TEST-SWH-2",
			"REQ-TEST-SWL-2 -> REQ-TEST-SWH-1",
			"REQ-TEST-SWL-2 -> REQ-TEST-SWH-2",
			"REQ-TEST-SWL-3 -> NIL",
		},
		matrixRows(rg, createUpstreamMatrix(rg, swlReqSpec, swhReqSpec)))
}
