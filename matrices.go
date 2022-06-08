/*
Functions which generate trace matrix tables between different levels of requirements and source code.
*/

package main

import (
	"fmt"
	"html/template"
	"io"
	"sort"

	"github.com/daedaleanai/reqtraq/config"
)

// GenerateTraceTables generates HTML for inspecting the gaps in the mappings between the two specified node types.
// @llr REQ-TRAQ-SWL-14, REQ-TRAQ-SWL-15
func (rg ReqGraph) GenerateTraceTables(w io.Writer, nodeTypeA, nodeTypeB config.ReqSpec) error {
	data := struct {
		From, To         string
		ItemsAB, ItemsBA []TableRow
	}{
		From: nodeTypeA.ToString(),
		To:   nodeTypeB.ToString(),
	}

	data.ItemsAB = rg.createDownstreamMatrix(nodeTypeA, nodeTypeB)
	data.ItemsBA = rg.createUpstreamMatrix(nodeTypeB, nodeTypeA)

	rg.sortMatrices(data.ItemsAB, data.ItemsBA)
	return matrixTmpl.ExecuteTemplate(w, "MATRIX", data)
}

func (rg ReqGraph) GenerateCodeTraceTables(w io.Writer, reqSpec config.ReqSpec) error {
	data := struct {
		From, To         string
		ItemsAB, ItemsBA []TableRow
	}{
		From: reqSpec.ToString(),
		To:   "CODE",
	}

	data.ItemsAB = rg.createSWLCodeMatrix(reqSpec)
	data.ItemsBA = rg.createCodeSWLMatrix(reqSpec)

	rg.sortMatrices(data.ItemsAB, data.ItemsBA)
	return matrixTmpl.ExecuteTemplate(w, "MATRIX", data)
}

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

// TableCell is a cell in a two-columns matrix, it can be a requirement or a code function.
type TableCell struct {
	Name        string // Name represents this item in the matrix.
	OrderNumber int    // OrderNumber can be used to order the items in a column ascending.
	req         *Req   // req is the represented requirement.
	code        *Code  // code is the represented code tag.
}

// TableRow is a pair of TableCell
type TableRow [2]*TableCell

// newCodeTableCell creates a new matrix cell from a code item
// @llr REQ-TRAQ-SWL-15
func newCodeTableCell(code *Code) *TableCell {
	item := &TableCell{}
	item.Name = fmt.Sprintf("%s: %s - %s", code.CodeFile.RepoName, code.CodeFile.Path, code.Tag)
	item.code = code
	return item
}

// newReqTableCell create a new matrix cell from a requirement item
// @llr REQ-TRAQ-SWL-14, REQ-TRAQ-SWL-15
func newReqTableCell(req *Req) *TableCell {
	item := &TableCell{}
	item.Name = req.ID
	item.req = req
	return item
}

// CodeOrderInfo contains everything needed to set the order number of a TableCell containing a code item.
type CodeOrderInfo struct {
	// filesIndex maps the code filename to an index of it's order alphabetically
	filesIndex map[string]int
	// fileIndexFactor holds the maximum line number of any function in any file
	fileIndexFactor int
}

// codeOrderInfo returns the info needed for sorting TableCells by code.
// @llr REQ-TRAQ-SWL-44
func (rg ReqGraph) codeOrderInfo() (info CodeOrderInfo) {
	info.filesIndex = make(map[string]int, len(rg.CodeTags))
	files := make([]string, 0, len(rg.CodeTags))
	info.fileIndexFactor = 0
	// build a list of filenames and find the function with the highest line number
	for file, tags := range rg.CodeTags {
		files = append(files, file.String())
		for _, codeTag := range tags {
			if info.fileIndexFactor < codeTag.Line {
				info.fileIndexFactor = codeTag.Line
			}
		}
	}
	// sort the filenames and store the indexes
	sort.Strings(files)
	for i, file := range files {
		info.filesIndex[file] = i
	}
	info.fileIndexFactor++
	return
}

// createCodeSWLMatrix creates an upstream matrix mapping code procedures to low level requirements.
// @llr REQ-TRAQ-SWL-15
func (rg *ReqGraph) createCodeSWLMatrix(reqSpec config.ReqSpec) []TableRow {
	items := make([]TableRow, 0)
	for _, tags := range rg.CodeTags {
		for _, codeTag := range tags {
			count := 0
			for _, parentReq := range codeTag.Parents {
				if parentReq.Document.MatchesSpec(reqSpec) {
					row := TableRow{newCodeTableCell(codeTag), newReqTableCell(parentReq)}
					items = append(items, row)
					count++
				}
			}
			if count == 0 {
				row := TableRow{newCodeTableCell(codeTag), nil}
				items = append(items, row)
			}
		}
	}
	return items
}

// createDownstreamMatrix returns a Trace Matrix from a set of requirements to a lower level set of requirements.
// @llr REQ-TRAQ-SWL-14
func (rg ReqGraph) createDownstreamMatrix(from, to config.ReqSpec) []TableRow {
	reqsHigh := rg.reqsWithSpec(from)
	items := make([]TableRow, 0, len(reqsHigh))
	for _, r := range reqsHigh {
		count := 0
		for _, childReq := range r.Children {
			if childReq.Document.MatchesSpec(to) {
				row := TableRow{newReqTableCell(r), newReqTableCell(childReq)}
				items = append(items, row)
				count++
			}
		}
		if count == 0 {
			row := TableRow{newReqTableCell(r), nil}
			items = append(items, row)
		}
	}
	return items
}

// createSWLCodeMatrix creates a downstream matrix mapping low level requirements to code procedures.
// @llr REQ-TRAQ-SWL-15
func (rg *ReqGraph) createSWLCodeMatrix(reqSpec config.ReqSpec) []TableRow {
	reqs := rg.reqsWithSpec(reqSpec)

	items := make([]TableRow, 0, len(reqs))
	for _, r := range reqs {
		count := 0
		for _, codeTag := range r.Tags {
			row := TableRow{newReqTableCell(r), newCodeTableCell(codeTag)}
			items = append(items, row)
			count++
		}
		if count == 0 {
			row := TableRow{newReqTableCell(r), nil}
			items = append(items, row)
		}
	}
	return items
}

// createUpstreamMatrix returns a Trace Matrix from a set of requirements to an upper level set of requirements.
// @llr REQ-TRAQ-SWL-14
func (rg ReqGraph) createUpstreamMatrix(from, to config.ReqSpec) []TableRow {
	reqsLow := rg.reqsWithSpec(from)
	items := make([]TableRow, 0, len(reqsLow))
	for _, r := range reqsLow {
		count := 0
		for _, parentReq := range r.Parents {
			if parentReq.Document.MatchesSpec(to) {
				row := TableRow{newReqTableCell(r), newReqTableCell(parentReq)}
				items = append(items, row)
				count++
			}
		}
		if count == 0 {
			row := TableRow{newReqTableCell(r), nil}
			items = append(items, row)
		}
	}
	return items
}

// reqsOfLevel returns the non-deleted requirements of the specified level, mapped by ID.
// @llr REQ-TRAQ-SWL-14
func (rg ReqGraph) reqsWithSpec(spec config.ReqSpec) map[string]*Req {
	reqs := make(map[string]*Req, 0)
	for _, r := range rg.Reqs {
		if r.Document.MatchesSpec(spec) && !r.IsDeleted() {
			reqs[r.ID] = r
		}
	}
	return reqs
}

// sortMatrices prepares the sort info and sorts the specified matrices.
// @llr REQ-TRAQ-SWL-42, REQ-TRAQ-SWL-43, REQ-TRAQ-SWL-44
func (rg ReqGraph) sortMatrices(matrices ...[]TableRow) {
	codeOrderInfo := rg.codeOrderInfo()
	for _, matrix := range matrices {
		// Update each item's OrderNumber.
		for _, row := range matrix {
			for _, item := range row {
				if item != nil {
					// Updated order number
					if item.req != nil {
						// requirements sorted by ID number
						item.OrderNumber = item.req.IDNumber
					} else if item.code != nil {
						if fileIdx, ok := codeOrderInfo.filesIndex[item.code.CodeFile.String()]; ok {
							item.OrderNumber = fileIdx*codeOrderInfo.fileIndexFactor + item.code.Line
						} else {
							panic("Code file could not be found in filesIndex. This is a bug")
						}
					} else {
						panic("Matrix element with no valid code or requirements. This should never happen")
					}
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
