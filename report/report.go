/*
Functions for generating HTML reports showing trace data.
*/

package report

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os/exec"

	"github.com/daedaleanai/reqtraq/code"
	"github.com/daedaleanai/reqtraq/reqs"
)

type reportData struct {
	Reqs   reqs.ReqGraph
	Filter *reqs.ReqFilter
	Once   Oncer
}

// ReportDown generates a HTML report of top down trace information.
// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-39
func ReportDown(rg *reqs.ReqGraph, w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "TOPDOWN", reportData{*rg, nil, Oncer{}})
}

// ReportUp generates a HTML report of bottom up trace information.
// @llr REQ-TRAQ-SWL-13, REQ-TRAQ-SWL-39
func ReportUp(rg *reqs.ReqGraph, w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "BOTTOMUP", reportData{*rg, nil, Oncer{}})
}

// ReportIssues generates a HTML report showing attribute and trace errors.
// @llr REQ-TRAQ-SWL-30, REQ-TRAQ-SWL-39
func ReportIssues(rg *reqs.ReqGraph, w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "ISSUES", reportData{*rg, nil, Oncer{}})
}

// ReportDownFiltered generates a HTML report of top down trace information, which has been filtered by the supplied parameters.
// @llr REQ-TRAQ-SWL-20, REQ-TRAQ-SWL-39
func ReportDownFiltered(rg *reqs.ReqGraph, w io.Writer, f *reqs.ReqFilter) error {
	return reportTmpl.ExecuteTemplate(w, "TOPDOWNFILT", reportData{*rg, f, Oncer{}})
}

// ReportUpFiltered generates a HTML report of bottom up trace information, which has been filtered by the supplied parameters.
// @llr REQ-TRAQ-SWL-21, REQ-TRAQ-SWL-39
func ReportUpFiltered(rg *reqs.ReqGraph, w io.Writer, f *reqs.ReqFilter) error {
	return reportTmpl.ExecuteTemplate(w, "BOTTOMUPFILT", reportData{*rg, f, Oncer{}})
}

// ReportIssuesFiltered generates a HTML report showing attribute and trace errors, which has been filtered by the supplied parameters.
// @llr REQ-TRAQ-SWL-31, REQ-TRAQ-SWL-39
func ReportIssuesFiltered(rg *reqs.ReqGraph, w io.Writer, f *reqs.ReqFilter) error {
	// TODO apply filter in ISSUESFILT template
	return reportTmpl.ExecuteTemplate(w, "ISSUESFILT", reportData{*rg, f, Oncer{}})
}

// Prints a filter in a nicely formatted manner to be shown in the report
// @llr REQ-TRAQ-SWL-19
func (report reportData) PrintFilter() string {
	if report.Filter != nil {
		filterString := ""
		if report.Filter.TitleRegexp != nil {
			filterString = fmt.Sprintf("%s (Title: \"%s\")", filterString, report.Filter.TitleRegexp)
		}
		if report.Filter.IDRegexp != nil {
			filterString = fmt.Sprintf("%s (ID: \"%s\")", filterString, report.Filter.IDRegexp)
		}
		if report.Filter.BodyRegexp != nil {
			filterString = fmt.Sprintf("%s (Body: \"%s\")", filterString, report.Filter.BodyRegexp)
		}
		if report.Filter.AnyAttributeRegexp != nil {
			filterString = fmt.Sprintf("%s (Any Attribute: \"%s\")", filterString, report.Filter.AnyAttributeRegexp)
		}
		if report.Filter.TitleRegexp != nil {
			filterString = fmt.Sprintf("%s (Attributes: \"%v\")", filterString, report.Filter.AttributeRegexp)
		}
		return filterString
	}
	return "No filter"
}

type Oncer map[string]bool

// Once maintains a map of requirements that have already been seen, if a requirement is seen multiple times
// subsequent occurrences are replaced with links to the first occurrence
// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func (o Oncer) Once(r *reqs.Req) *reqs.Req {
	ok := o[r.ID]
	o[r.ID] = true
	if !ok {
		return r
	}
	return &reqs.Req{ID: r.ID, Title: r.Title, Body: r.Body, Document: nil}
}

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

// formatBodyAsHTML converts a string containing markdown to HTML using pandoc.
// @llr REQ-TRAQ-SWL-41
func formatBodyAsHTML(txt string) template.HTML {
	cmd := exec.Command("pandoc", "--mathjax")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("Couldn't get input pipe for pandoc: ", err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, txt)
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal("Error while running pandoc: ", err)
	}

	return template.HTML(out)
}

var functionMap = template.FuncMap{
	"formatBodyAsHTML": formatBodyAsHTML,
	"codeFileToString": codeFileToString,
	"isImpl":           isImpl,
	"isTest":           isTest,
	"shouldShowTag":    shouldShowTag,
	"listCodeParents":  listCodeParents,
}
var reportTmpl = template.Must(template.Must(template.New("").Funcs(functionMap).Parse(headerFooterTmplText)).Parse(reportTmplText))

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func codeFileToString(CodeFile code.CodeFile) string {
	return CodeFile.String()
}

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func isImpl(CodeFile code.CodeFile) bool {
	return CodeFile.Type.Matches(code.CodeTypeImplementation)
}

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func isTest(CodeFile code.CodeFile) bool {
	return CodeFile.Type.Matches(code.CodeTypeTests)
}

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func shouldShowTag(code *code.Code, rg reqs.ReqGraph) bool {
	return !code.Optional || (len(listCodeParents(code.Links, rg)) != 0)
}

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func listCodeParents(links []code.ReqLink, rg reqs.ReqGraph) []*reqs.Req {
	var parents []*reqs.Req
	for _, link := range links {
		if parent, ok := rg.Reqs[link.Id]; ok {
			parents = append(parents, parent)
		}
	}
	return parents
}

var reportTmplText = `
{{ define "REQUIREMENT" }}
	{{if ne .Document nil }}
		<h3><a name="{{ .ID }}"></a>{{ .ID }} {{ .Title }}</h3>
		{{ if .Body }}
			<p>{{formatBodyAsHTML .Body }}</p>
		{{ end }}
		{{ if .Attributes }}
			<ul style="list-style: none; padding: 0; margin: 0;">
			{{ range $k, $v := .Attributes }}
				<li><strong>{{ $k }}</strong>: {{ $v }}</li>
			{{ end }}
			</ul>
		{{ end }}
	{{ else }}
		<h3><a href="#{{ .ID }}">{{ .ID }} {{ .Title }}</a></h3>
 	{{end}}
{{ end }}

{{ define "CODETAGS"}}
	{{ if . }}
		<p>Code Implementation:
		{{ range . }}
			{{ if isImpl .CodeFile }}
				<a href="{{ .URL }}" target="_blank">{{ codeFileToString .CodeFile }} - {{ .Tag }}</a>
			{{ end }}
		{{ end }}
		</p>
		<p>Code Tests:
		{{ range . }}
			{{ if isTest .CodeFile }}
				<a href="{{ .URL }}" target="_blank">{{ codeFileToString .CodeFile }} - {{ .Tag }}</a>
			{{ end }}
		{{ end }}
		</p>
	{{ end }}
{{ end }}

{{ define "CHANGELIST" }}
	{{ if . }}
		<p>Changelists:
			{{ range $k, $v := . }}
				<a href="{{ $v }}" target="_blank"><span class="label label-primary">{{ $k }}</span></a>
			{{ end }}
		</p>
	{{ end }}
{{ end }}

{{define "TOPDOWN"}}
	{{template "HEADER"}}
	<h1>Top Down Tracing</h1>

	<ul style="list-style: none; padding: 0; margin: 0;">
		{{ range .Reqs.OrdsByPosition }}
			<li>
				{{ template "REQUIREMENT" . }}
				<!-- HLRs -->
				<ul>
				{{ range .Children }}
					<li>
						{{ template "REQUIREMENT" ($.Once.Once .) }}
						<!-- LLRs -->
							<ul>
							{{ range .Children }}
								<li>
									{{ with ($.Once.Once .) }}
										{{ template "REQUIREMENT" . }}
										{{ template "CODETAGS" .Tags }}
										{{ template "CHANGELIST" .Changelists }}
									{{ end }}
								</li>
							{{ else }}
								<li class="text-danger">No children</li>
							{{ end }}
							</ul>
					</li>
					{{ else }}
						<li class="text-danger">No children</li>
					{{ end }}
				</ul>
			</li>
		{{ else }}
			<li  class="text-danger">Empty graph</li>
		{{ end }}
	</ul>
	{{template "FOOTER"}}
{{end}}

{{define "BOTTOMUP"}}
	{{template "HEADER"}}
	<h1>Bottom Up Tracing</h1>

	<ul style="list-style: none; padding: 0; margin: 0;">
		{{ range .Reqs.CodeTags }}
		{{ range . }}
		{{ if shouldShowTag . $.Reqs }}
			<li>
				{{ if isImpl .CodeFile }}
					<h3><a href="{{ .URL }}" target="_blank">Impl: {{ codeFileToString .CodeFile }} - {{ .Tag }}</a></h3>
				{{ else }}
					<h3><a href="{{ .URL }}" target="_blank">Test: {{ codeFileToString .CodeFile }} - {{ .Tag }}</a></h3>
				{{ end }}

				<!-- LLRs -->
				<ul>
					{{ range listCodeParents .Links $.Reqs }}
					{{ with ($.Once.Once .) }}
					<li>
						{{ template "REQUIREMENT" . }}
						{{ template "CHANGELIST" .Changelists }}

						<!-- HLRs -->
						<ul>
							{{ range .Parents }}
							{{ with ($.Once.Once .) }}
							<li>
								{{ template "REQUIREMENT" . }}

								<!-- SYSTEM -->
								<ul>
									{{ range .Parents }}
									{{ with ($.Once.Once .) }}
									<li>
										{{ template "REQUIREMENT" . }}
									</li>
									{{ end }}
									{{ else }}
										<li class="text-danger">No parents</li>
									{{ end }}
								</ul>
							</li>
							{{ end }}
							{{ else }}
								<li class="text-danger">No parents</li>
							{{ end }}
						</ul>
					</li>
					{{ end }}
					{{ else }}
						<li class="text-danger">No parents</li>
					{{ end }}
				</ul>
			</li>
		{{ end }}
		{{ end }}
		{{ else }}
			<li class="text-danger">Empty graph</li>
		{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}

{{ define "ISSUES" }}
	{{template "HEADER"}}
	<h1>Issues</h1>

	<ul>
	{{ range .Reqs.Issues }}
		<li>
			{{ .Description }}
		</li>
	{{ else }}
		<li class="text-success">No basic errors found.</li>
	{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}

{{ define "TOPDOWNFILT"}}
	{{template "HEADER"}}
	<h1>Top Down Tracing</h1>

	<h3><em>Filter Criteria: {{ .PrintFilter }} </em></h3>
	<ul style="list-style: none; padding: 0; margin: 0;">
		{{ range .Reqs.OrdsByPosition }}
			{{ if .Matches $.Filter }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
			{{ range .Children }}
				{{ if .Matches $.Filter }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
				{{ range .Children }}
					{{ if .Matches $.Filter }}
						{{ with ($.Once.Once .) }}
							{{ template "REQUIREMENT" . }}
							{{ template "CODETAGS" .Tags }}
							{{ template "CHANGELIST" .Changelists }}
						{{ end }}
					{{ end }}
				{{ end }}
			{{ end }}
		{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}

{{ define "BOTTOMUPFILT" }}
	{{template "HEADER" }}
	<h1>Bottom Up Tracing</h1>

	<h3><em>Filter Criteria: {{ .PrintFilter }} </em></h3>
	<ul style="list-style: none; padding: 0; margin: 0;">
		{{ range .Reqs.CodeTags }}
		{{ range . }}
		{{ if shouldShowTag . $.Reqs }}
			{{ range listCodeParents .Links $.Reqs }}
				{{ if .Matches $.Filter }}
					{{ with ($.Once.Once .) }}
						{{ template "REQUIREMENT" . }}
						{{ template "CODETAGS" .Tags }}
						{{ template "CHANGELIST" .Changelists }}
					{{ end }}
				{{ end }}
				{{ range .Parents }}
					{{ if .Matches $.Filter }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
						{{ range .Parents }}
							{{ if .Matches $.Filter }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
						{{ end }}
				{{ end }}
			{{ end }}
		{{ end }}
		{{ end }}
		{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}

{{ define "ISSUESFILT" }}
	{{template "HEADER"}}
	<h1>Issues</h1>

	<h3><em>Filter Criteria: {{ .PrintFilter }} </em></h3>
	<ul>
	{{ range .Reqs.Issues }}
		<li>
			{{ .Description }}
		</li>
	{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}
`
