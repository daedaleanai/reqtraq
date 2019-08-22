package main

import (
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
)

var matrixTmpl = template.Must(template.Must(template.New("").Parse(headerFooterTmplText)).Parse(matrixTmplText))

var matrixTmplText = `
{{ define "MATRIXTABLE" }}
<div class="trace-matrix-table">
{{- range . }}
	<div>
	{{- range . }}
		{{ if . -}}
			<div>{{ .ID }}</div>
		{{- else -}}
			<div class="hole"></div>
		{{- end -}}
	{{ end }}
	</div>
{{- end -}}
</div>
{{ end }}


{{ define "MATRIX" }}
	{{template "HEADER"}}
	<h1>Trace Matrices {{ .From }}&ndash;{{ .To }}</h1>

	<div style="display: table; padding-top: 1em;">
		<div style="display: table-row">
			<div style="display: table-cell">
				{{ template "MATRIXTABLE" .ItemsAB }}
			</div>
			<div style="display: table-cell; padding-left: 2em;">
				{{ template "MATRIXTABLE" .ItemsBA }}
			</div>
		</div>
	</div>

	{{ template "FOOTER" }}
{{ end }}
`

func (rg reqGraph) ReportHoles(w io.Writer, nodeTypeA, nodeTypeB string) error {
	levelA, ok := config.ReqTypeToReqLevel[nodeTypeA]
	if !ok {
		return fmt.Errorf("unknown node type: %s", nodeTypeA)
	}
	levelB, ok := config.ReqTypeToReqLevel[nodeTypeB]
	if !ok {
		return fmt.Errorf("unknown node type: %s", nodeTypeB)
	}

	data := struct {
		From, To         string
		ItemsAB, ItemsBA [][2]*Req
	}{
		From:    nodeTypeA,
		To:      nodeTypeB,
		ItemsAB: rg.createMatrix(levelA, levelB),
		ItemsBA: rg.createMatrix(levelB, levelA),
	}
	return matrixTmpl.ExecuteTemplate(w, "MATRIX", data)
}

func (rg reqGraph) createMatrix(from, to config.RequirementLevel) [][2]*Req {
	fromReqs := make(map[string]*Req, 0)
	for _, r := range rg.Reqs {
		if r.Level == from {
			fromReqs[r.ID] = r
		}
	}

	var items [][2]*Req
	// Higher levels are lower in value.
	if from < to {
		items = createDownstreamMatrix(fromReqs, to)
	} else {
		items = createUpstreamMatrix(fromReqs, to)
	}

	sort.Slice(items, func(i, j int) bool {
		a, b := items[i][0], items[j][0]
		if a.IDNumber != b.IDNumber {
			return a.IDNumber < b.IDNumber
		}
		res := strings.Compare(a.ID, b.ID)
		if res == 0 {
			a, b := items[i][1], items[j][1]
			if a.IDNumber != b.IDNumber {
				return a.IDNumber < b.IDNumber
			}
			return -1 == strings.Compare(a.ID, b.ID)
		}
		return -1 == res
	})

	return items
}

// createDownstreamMatrix returns a Trace Matrix from a set of nodes to
// a lower level set of nodes.
func createDownstreamMatrix(reqsHigh map[string]*Req, toLevel config.RequirementLevel) [][2]*Req {
	items := make([][2]*Req, 0, len(reqsHigh))
	for _, r := range reqsHigh {
		count := 0
		for _, childReq := range r.Children {
			if childReq.Level == toLevel {
				row := [2]*Req{r, childReq}
				items = append(items, row)
				count++
			}
		}
		if count == 0 {
			row := [2]*Req{r, nil}
			items = append(items, row)
		}
	}
	return items
}

// createUpstreamMatrix returns a Trace Matrix from a set of nodes to
// an upper level set of nodes.
func createUpstreamMatrix(reqsLow map[string]*Req, toLevel config.RequirementLevel) [][2]*Req {
	items := make([][2]*Req, 0, len(reqsLow))
	for _, r := range reqsLow {
		count := 0
		for _, parentReq := range r.Parents {
			if parentReq.Level == toLevel {
				row := [2]*Req{r, parentReq}
				items = append(items, row)
				count++
			}
		}
		if count == 0 {
			row := [2]*Req{r, nil}
			items = append(items, row)
		}
	}
	return items
}
