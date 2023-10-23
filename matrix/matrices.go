/*
Functions which generate trace matrix tables between different levels of requirements and source code.
*/

package matrix

import (
	"fmt"
	"html/template"
	"io"
	"sort"

	"github.com/daedaleanai/reqtraq/code"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/reqs"
)

var headerFooterTmplText = `
{{define "HEADER"}}
<html lang="en">
	<head>
		<meta charset="utf-8">
	    <meta http-equiv="X-UA-Compatible" content="IE=edge">
	    <meta name="viewport" content="width=device-width, initial-scale=1">
	    <meta name="description" content="">
	    <meta name="author" content="">

		<title>Reqtraq - Daedalean AG</title>

		<!-- BOOTSTRAP -->
		<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">

		<!-- CUSTOM -->
		<style>
			h1 {
				text-align: left;
			}
			body {
				font-family: Roboto, Arial, sans-serif;
				max-width: 3000px;
				margin-left: 5%;
				margin-right: 5%;
			}
			a, a:hover {
				text-decoration: none;
			}
			div.trace-matrix-table {
				display: table;
				border: 1px solid black
			}
			div.trace-matrix-table > div {
				display: table-row;
			}
			div.trace-matrix-table > div > div {
				display: table-cell;
				padding: 0em 0.5em;
			}
		</style>
		<!-- Load MathJax for rendering of equations -->
		<script type="text/javascript" async
			src="https://cdnjs.cloudflare.com/ajax/libs/mathjax/2.7.1/MathJax.js?config=TeX-AMS-MML_HTMLorMML">
		</script>

	</head>
	<body>
{{end}}

{{define "FOOTER"}}
	</body>
</html>
{{end}}
`

// GenerateTraceTables generates HTML for inspecting the gaps in the mappings between the two specified node types.
// @llr REQ-TRAQ-SWL-14
func GenerateTraceTables(rg *reqs.ReqGraph, w io.Writer, nodeTypeA, nodeTypeB config.ReqSpec) error {
	data := struct {
		From, To         string
		ItemsAB, ItemsBA []TableRow
	}{
		From: nodeTypeA.String(),
		To:   nodeTypeB.String(),
	}

	data.ItemsAB = createDownstreamMatrix(rg, nodeTypeA, nodeTypeB)
	data.ItemsBA = createUpstreamMatrix(rg, nodeTypeB, nodeTypeA)

	sortMatrices(rg, data.ItemsAB, data.ItemsBA)
	return matrixTmpl.ExecuteTemplate(w, "MATRIX", data)
}

// GenerateCodeTraceTables generates HTML for inspecting the gaps in the mappings between the specified
// node type and code
// @llr REQ-TRAQ-SWL-15, REQ-TRAQ-SWL-71, REQ-TRAQ-SWL-72
func GenerateCodeTraceTables(rg *reqs.ReqGraph, w io.Writer, reqSpec config.ReqSpec, codeType code.CodeType) error {
	data := struct {
		From, To         string
		ItemsAB, ItemsBA []TableRow
	}{
		From: reqSpec.String(),
		To:   codeType.String(),
	}

	data.ItemsAB = createSWLCodeMatrix(rg, reqSpec, codeType)
	data.ItemsBA = createCodeSWLMatrix(rg, reqSpec, codeType)

	sortMatrices(rg, data.ItemsAB, data.ItemsBA)
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
	<h1>Trace Matrices {{ .From }} &ndash; {{ .To }}</h1>

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
	Name        string     // Name represents this item in the matrix.
	OrderNumber int        // OrderNumber can be used to order the items in a column ascending.
	req         *reqs.Req  // req is the represented requirement.
	code        *code.Code // code is the represented code tag.
}

// TableRow is a pair of TableCell
type TableRow [2]*TableCell

// newCodeTableCell creates a new matrix cell from a code item
// @llr REQ-TRAQ-SWL-15
func newCodeTableCell(code *code.Code) *TableCell {
	item := &TableCell{}
	item.Name = fmt.Sprintf("%s: %s - %s", code.CodeFile.RepoName, code.CodeFile.Path, code.Tag)
	item.code = code
	return item
}

// newReqTableCell create a new matrix cell from a requirement item
// @llr REQ-TRAQ-SWL-14, REQ-TRAQ-SWL-15
func newReqTableCell(req *reqs.Req) *TableCell {
	item := &TableCell{}
	item.Name = req.ID
	item.req = req
	return item
}

// CodeOrderInfo contains everything needed to set the order number of a
// TableCell mapping a code item. We need to be able to order the code items
// first by repo and file name alphabetically and finally by line number.
type CodeOrderInfo struct {
	// filesIndex maps the code file to an index of it's order alphabetically.
	filesIndex map[code.CodeFile]int
	// fileIndexFactor holds the maximum line number of any function in any
	// file plus one.
	fileIndexFactor int
}

// codeOrderInfo returns the info needed for sorting TableCells by code.
// @llr REQ-TRAQ-SWL-44
func codeOrderInfo(rg *reqs.ReqGraph) (info CodeOrderInfo) {
	filesSet := make(map[code.CodeFile]bool)
	info.fileIndexFactor = 0
	// build a list of filenames and find the function with the highest line number
	for _, tags := range rg.CodeTags {
		for _, codeTag := range tags {
			filesSet[codeTag.CodeFile] = true
			hasParents := false
			for _, link := range codeTag.Links {
				if _, ok := rg.Reqs[link.Id]; ok {
					hasParents = true
					break
				}
			}
			if codeTag.Optional && !hasParents {
				// This code does not link to any requirement and it doesn't
				// even have to, so we ignore it.
				continue
			}
			if info.fileIndexFactor < codeTag.Line {
				info.fileIndexFactor = codeTag.Line
			}
		}
	}
	info.fileIndexFactor++

	// sort the filenames and store the indexes
	files := make([]code.CodeFile, 0, len(filesSet))
	for repoFile := range filesSet {
		files = append(files, repoFile)
	}
	sort.Slice(files, func(i, j int) bool {
		a := &files[i]
		b := &files[j]
		return a.RepoName < b.RepoName || (a.RepoName == b.RepoName && a.Path < b.Path)
	})

	info.filesIndex = make(map[code.CodeFile]int)
	for i, f := range files {
		info.filesIndex[f] = i
	}

	return
}

// createCodeSWLMatrix creates an upstream matrix mapping code procedures to low level requirements.
// @llr REQ-TRAQ-SWL-15, REQ-TRAQ-SWL-71, REQ-TRAQ-SWL-72,
func createCodeSWLMatrix(rg *reqs.ReqGraph, reqSpec config.ReqSpec, codeType code.CodeType) []TableRow {
	items := make([]TableRow, 0)
	for _, tags := range rg.CodeTags {
		for _, codeTag := range tags {
			if !codeTag.CodeFile.Type.Matches(codeType) {
				continue
			}

			count := 0
			for _, parentLink := range codeTag.Links {
				if parentReq, ok := rg.Reqs[parentLink.Id]; ok {
					if parentReq.Document.MatchesSpec(reqSpec) {
						row := TableRow{newCodeTableCell(codeTag), newReqTableCell(parentReq)}
						items = append(items, row)
						count++
					}
				}
			}
			if count == 0 && !codeTag.Optional {
				// This code procedure does not link to any requirement matching
				// the reqSpec. Display it with a gap.
				row := TableRow{newCodeTableCell(codeTag), nil}
				items = append(items, row)
			}
		}
	}
	return items
}

// createDownstreamMatrix returns a Trace Matrix from a set of requirements to a lower level set of requirements.
// @llr REQ-TRAQ-SWL-14
func createDownstreamMatrix(rg *reqs.ReqGraph, from, to config.ReqSpec) []TableRow {
	reqsHigh := reqsWithSpec(rg, from)
	items := make([]TableRow, 0, len(reqsHigh))
	for _, r := range reqsHigh {
		count := 0
		for _, childReq := range r.Children {
			if childReq.Document.MatchesSpec(to) && to.Re.MatchString(childReq.ID) {
				if to.AttrKey != "" && !to.AttrVal.MatchString(childReq.Attributes[to.AttrKey]) {
					continue
				}
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
// @llr REQ-TRAQ-SWL-15, REQ-TRAQ-SWL-71, REQ-TRAQ-SWL-72,
func createSWLCodeMatrix(rg *reqs.ReqGraph, reqSpec config.ReqSpec, codeType code.CodeType) []TableRow {
	reqs := reqsWithSpec(rg, reqSpec)

	items := make([]TableRow, 0, len(reqs))
	for _, r := range reqs {
		count := 0
		for _, codeTag := range r.Tags {
			if !codeTag.CodeFile.Type.Matches(codeType) {
				continue
			}

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
func createUpstreamMatrix(rg *reqs.ReqGraph, from, to config.ReqSpec) []TableRow {
	reqsLow := reqsWithSpec(rg, from)
	items := make([]TableRow, 0, len(reqsLow))
	for _, r := range reqsLow {
		count := 0
		for _, parentReq := range r.Parents {
			if parentReq.Document.MatchesSpec(to) && to.Re.MatchString(parentReq.ID) {
				if to.AttrKey != "" && !to.AttrVal.MatchString(parentReq.Attributes[to.AttrKey]) {
					continue
				}
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

// reqsWithSpec returns the non-deleted requirements of the specified ReqSpec, mapped by ID.
// @llr REQ-TRAQ-SWL-14
func reqsWithSpec(rg *reqs.ReqGraph, spec config.ReqSpec) map[string]*reqs.Req {
	reqs := make(map[string]*reqs.Req, 0)
	for _, r := range rg.Reqs {
		if r.Document.MatchesSpec(spec) && spec.Re.MatchString(r.ID) && !r.IsDeleted() {
			if spec.AttrKey != "" && !spec.AttrVal.MatchString(r.Attributes[spec.AttrKey]) {
				continue
			}
			reqs[r.ID] = r
		}
	}
	return reqs
}

// sortMatrices prepares the sort info and sorts the specified matrices.
// @llr REQ-TRAQ-SWL-42, REQ-TRAQ-SWL-43, REQ-TRAQ-SWL-44
func sortMatrices(rg *reqs.ReqGraph, matrices ...[]TableRow) {
	codeOrderInfo := codeOrderInfo(rg)
	for _, matrix := range matrices {
		for _, row := range matrix {
			// We calculate the OrderNumber of both cells of the row because we
			// display the rows both A->B and B->A.
			for _, item := range row {
				if item != nil {
					if item.req != nil {
						// When a column has requirement IDs, we assume all of
						// them originate in the same certdoc. Then we can
						// simply order them by requirement numerical ID.
						item.OrderNumber = item.req.IDNumber
					} else if item.code != nil {
						// When a column has code procedures, we need to order
						// them first by file and then by line number.
						if fileIdx, ok := codeOrderInfo.filesIndex[item.code.CodeFile]; ok {
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
			if a1 == nil {
				return true
			} else if b1 == nil {
				return false
			}
			return a1.OrderNumber < b1.OrderNumber
		})
	}
}
