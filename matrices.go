package main

import (
	"fmt"
	"html/template"
	"io"
	"sort"

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
			<div>{{ .Name }}</div>
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

// MatrixItem is a cell in a two-columns matrix.
type MatrixItem struct {
	// Name represents this item in the matrix.
	Name string
	// OrderNumber can be used to order the items in a column ascending.
	OrderNumber int
	// req is the represented requirement.
	req *Req
	// code is the represented code tag.
	code *Code
}

func (item *MatrixItem) UpdateOrderNumber(info OrderInfo) {
	if item.req != nil {
		item.OrderNumber = item.req.IDNumber
	} else if item.code != nil {
		item.OrderNumber = info.filesIndex[item.code.Path]*info.fileIndexFactor + item.code.Line
	} else {
		item.OrderNumber = 0
	}
}

func NewReqMatrixItem(req *Req) *MatrixItem {
	item := &MatrixItem{}
	item.Name = req.ID
	item.req = req
	return item
}

func NewCodeMatrixItem(code *Code) *MatrixItem {
	item := &MatrixItem{}
	item.Name = code.Path + " " + code.Tag
	item.code = code
	return item
}

// ReportHoles generates HTML for inspecting the gaps in the mappings
// between the two specified node types.
func (rg reqGraph) ReportHoles(w io.Writer, nodeTypeA, nodeTypeB string) error {
	data := struct {
		From, To         string
		ItemsAB, ItemsBA [][2]*MatrixItem
	}{
		From: nodeTypeA,
		To:   nodeTypeB,
	}

	switch nodeTypeA + ":" + nodeTypeB {
	case "SYS:SWH":
		data.ItemsAB = rg.createDownstreamMatrix(config.SYSTEM, config.HIGH)
		data.ItemsBA = rg.createUpstreamMatrix(config.HIGH, config.SYSTEM)
	case "SWH:SWL":
		data.ItemsAB = rg.createDownstreamMatrix(config.HIGH, config.LOW)
		data.ItemsBA = rg.createUpstreamMatrix(config.LOW, config.HIGH)
	case "SWL:CODE":
		data.ItemsAB = rg.createSWLCodeMatrix()
		data.ItemsBA = rg.createCodeSWLMatrix()
	default:
		return fmt.Errorf("unknown mapping: %s-%s", nodeTypeA, nodeTypeB)
	}

	rg.sortMatrices(data.ItemsAB, data.ItemsBA)
	return matrixTmpl.ExecuteTemplate(w, "MATRIX", data)
}

// OrderInfo contains everything needed to set the order number of a MatrixItem.
type OrderInfo struct {
	// filesIndex contains the indexes determining the order of the code files.
	filesIndex map[string]int
	// fileIndexFactor is applied when calculating the order of a procedure:
	// fileIndex * fileIndexFactor + procedureLineNumber
	fileIndexFactor int
}

// CodeOrderInfo returns the info needed for sorting MatrixItems.
func (rg reqGraph) CodeOrderInfo() (info OrderInfo) {
	info.filesIndex = make(map[string]int, len(rg.CodeTags))
	files := make([]string, 0, len(rg.CodeTags))
	info.fileIndexFactor = 0
	for file, tags := range rg.CodeTags {
		files = append(files, file)
		for _, codeTag := range tags {
			if info.fileIndexFactor < codeTag.Line {
				info.fileIndexFactor = codeTag.Line
			}
		}
	}
	sort.Strings(files)
	info.fileIndexFactor++
	for i, file := range files {
		info.filesIndex[file] = i
	}
	return
}

// reqsOfLevel returns the non-deleted requirements of the specified level,
// mapped by ID.
func (rg reqGraph) reqsOfLevel(level config.RequirementLevel) map[string]*Req {
	reqs := make(map[string]*Req, 0)
	for _, r := range rg.Reqs {
		if r.Level == level && !r.IsDeleted() {
			reqs[r.ID] = r
		}
	}
	return reqs
}

// sortMatrices prepares the sort info and sorts the specified matrices.
func (rg reqGraph) sortMatrices(matrices ...[][2]*MatrixItem) {
	codeOrderInfo := rg.CodeOrderInfo()
	for _, matrix := range matrices {
		// Update each item's OrderNumber.
		for _, row := range matrix {
			for _, item := range row {
				if item != nil {
					item.UpdateOrderNumber(codeOrderInfo)
				}
			}
		}
		// Sorts the rows based on the OrderNumber of the items.
		sort.Slice(matrix, func(i, j int) bool {
			a0, b0 := matrix[i][0], matrix[j][0]
			if a0.OrderNumber != b0.OrderNumber {
				return a0.OrderNumber < b0.OrderNumber
			}
			a1, b1 := matrix[i][1], matrix[j][1]
			return a1.OrderNumber < b1.OrderNumber
		})
	}
}

// createDownstreamMatrix returns a Trace Matrix from a set of requirements to
// a lower level set of requirements.
func (rg reqGraph) createDownstreamMatrix(from, to config.RequirementLevel) [][2]*MatrixItem {
	reqsHigh := rg.reqsOfLevel(from)
	items := make([][2]*MatrixItem, 0, len(reqsHigh))
	for _, r := range reqsHigh {
		count := 0
		for _, childReq := range r.Children {
			if childReq.Level == to {
				row := [2]*MatrixItem{NewReqMatrixItem(r), NewReqMatrixItem(childReq)}
				items = append(items, row)
				count++
			}
		}
		if count == 0 {
			row := [2]*MatrixItem{NewReqMatrixItem(r), nil}
			items = append(items, row)
		}
	}
	return items
}

// createUpstreamMatrix returns a Trace Matrix from a set of requirements to
// an upper level set of requirements.
func (rg reqGraph) createUpstreamMatrix(from, to config.RequirementLevel) [][2]*MatrixItem {
	reqsLow := rg.reqsOfLevel(from)
	items := make([][2]*MatrixItem, 0, len(reqsLow))
	for _, r := range reqsLow {
		count := 0
		for _, parentReq := range r.Parents {
			if parentReq.Level == to {
				row := [2]*MatrixItem{NewReqMatrixItem(r), NewReqMatrixItem(parentReq)}
				items = append(items, row)
				count++
			}
		}
		if count == 0 {
			row := [2]*MatrixItem{NewReqMatrixItem(r), nil}
			items = append(items, row)
		}
	}
	return items
}

// createSWLCodeMatrix creates a downstream matrix mapping
// low level requirements to code procedures.
func (rg *reqGraph) createSWLCodeMatrix() [][2]*MatrixItem {
	reqs := rg.reqsOfLevel(config.LOW)
	items := make([][2]*MatrixItem, 0, len(reqs))
	for _, r := range reqs {
		count := 0
		for _, codeTag := range r.Tags {
			row := [2]*MatrixItem{NewReqMatrixItem(r), NewCodeMatrixItem(codeTag)}
			items = append(items, row)
			count++
		}
		if count == 0 {
			row := [2]*MatrixItem{NewReqMatrixItem(r), nil}
			items = append(items, row)
		}
	}
	return items
}

// createCodeSWLMatrix creates an upstream matrix mapping
// code procedures to low level requirements.
func (rg *reqGraph) createCodeSWLMatrix() [][2]*MatrixItem {

	items := make([][2]*MatrixItem, 0)
	for _, tags := range rg.CodeTags {
		for _, codeTag := range tags {
			count := 0
			for _, parentReq := range codeTag.Parents {
				row := [2]*MatrixItem{NewCodeMatrixItem(codeTag), NewReqMatrixItem(parentReq)}
				items = append(items, row)
				count++
			}
			if count == 0 {
				row := [2]*MatrixItem{NewCodeMatrixItem(codeTag), nil}
				items = append(items, row)
			}
		}
	}
	return items
}
