/*
Functions for generating HTML reports showing trace data.
*/

package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os/exec"
	"sort"
)

type reportData struct {
	Reqs   ReqGraph
	Filter *ReqFilter
	Once   Oncer
	Diffs  map[string][]string
}

// ReportDown generates a HTML report of top down trace information.
// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-39
func (rg ReqGraph) ReportDown(w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "TOPDOWN", reportData{rg, nil, Oncer{}, nil})
}

// ReportUp generates a HTML report of bottom up trace information.
// @llr REQ-TRAQ-SWL-13, REQ-TRAQ-SWL-39
func (rg ReqGraph) ReportUp(w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "BOTTOMUP", reportData{rg, nil, Oncer{}, nil})
}

// ReportIssues generates a HTML report showing attribute and trace errors.
// @llr REQ-TRAQ-SWL-30, REQ-TRAQ-SWL-39
func (rg ReqGraph) ReportIssues(w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "ISSUES", reportData{rg, nil, Oncer{}, nil})
}

// ReportDownFiltered generates a HTML report of top down trace information, which has been filtered by the supplied parameters.
// @llr REQ-TRAQ-SWL-20, REQ-TRAQ-SWL-39
func (rg ReqGraph) ReportDownFiltered(w io.Writer, f *ReqFilter, diffs map[string][]string) error {
	return reportTmpl.ExecuteTemplate(w, "TOPDOWNFILT", reportData{rg, f, Oncer{}, diffs})
}

// ReportUpFiltered generates a HTML report of bottom up trace information, which has been filtered by the supplied parameters.
// @llr REQ-TRAQ-SWL-21, REQ-TRAQ-SWL-39
func (rg ReqGraph) ReportUpFiltered(w io.Writer, f *ReqFilter, diffs map[string][]string) error {
	return reportTmpl.ExecuteTemplate(w, "BOTTOMUPFILT", reportData{rg, f, Oncer{}, diffs})
}

// ReportIssuesFiltered generates a HTML report showing attribute and trace errors, which has been filtered by the supplied parameters.
// @llr REQ-TRAQ-SWL-31, REQ-TRAQ-SWL-39
func (rg ReqGraph) ReportIssuesFiltered(w io.Writer, filter *ReqFilter, diffs map[string][]string) error {
	// TODO apply filter in ISSUESFILT template
	return reportTmpl.ExecuteTemplate(w, "ISSUESFILT", reportData{rg, filter, Oncer{}, diffs})
}

// OrdsByPosition returns the SYSTEM requirements which don't have any parent, ordered by position.
// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-20
func (rg ReqGraph) OrdsByPosition() []*Req {
	var r []*Req
	for _, v := range rg.Reqs {
		if !v.Document.HasParent() && len(v.ParentIds) == 0 {
			r = append(r, v)
		}
	}
	sort.Sort(byPosition(r))
	return r
}

// Matches returns true if the requirement matches the filter AND its ID is in the diffs map, if any.
// @llr REQ-TRAQ-SWL-19
func (r *Req) Matches(filter *ReqFilter, diffs map[string][]string) bool {
	if filter != nil {
		if filter.IDRegexp != nil {
			if !filter.IDRegexp.MatchString(r.ID) {
				return false
			}
		}
		if filter.TitleRegexp != nil {
			if !filter.TitleRegexp.MatchString(r.Title) {
				return false
			}
		}
		if filter.BodyRegexp != nil {
			if !filter.BodyRegexp.MatchString(r.Body) {
				return false
			}
		}
		if filter.AnyAttributeRegexp != nil {
			var matches bool
			// Any of the existing attributes must match.
			for _, value := range r.Attributes {
				if filter.AnyAttributeRegexp.MatchString(value) {
					matches = true
					break
				}
			}
			if !matches {
				return false
			}
		}
		// Each of the filtered attributes must match.
		for a, e := range filter.AttributeRegexp {
			if !e.MatchString(r.Attributes[a]) {
				return false
			}
		}
	}
	if diffs != nil {
		_, ok := diffs[r.ID]
		return ok
	}
	return true
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
func (o Oncer) Once(r *Req) *Req {
	ok := o[r.ID]
	o[r.ID] = true
	if !ok {
		return r
	}
	return &Req{ID: r.ID, Title: r.Title, Body: r.Body, Document: nil}
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

var reportTmpl = template.Must(template.Must(template.New("").Funcs(template.FuncMap{"formatBodyAsHTML": formatBodyAsHTML, "codeFileToString": codeFileToString, "isImpl": isImpl, "isTest": isTest, "shouldShowTag": shouldShowTag}).Parse(headerFooterTmplText)).Parse(reportTmplText))

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func codeFileToString(CodeFile CodeFile) string {
	return CodeFile.String()
}

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func isImpl(CodeFile CodeFile) bool {
	return CodeFile.Type.Matches(CodeTypeImplementation)
}

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func isTest(CodeFile CodeFile) bool {
	return CodeFile.Type.Matches(CodeTypeTests)
}

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13
func shouldShowTag(code *Code) bool {
	return !code.Optional || (len(code.Parents) != 0)
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
		{{ if shouldShowTag . }}
			<li>
				{{ if isImpl .CodeFile }}
					<h3><a href="{{ .URL }}" target="_blank">Impl: {{ codeFileToString .CodeFile }} - {{ .Tag }}</a></h3>
				{{ else }}
					<h3><a href="{{ .URL }}" target="_blank">Test: {{ codeFileToString .CodeFile }} - {{ .Tag }}</a></h3>
				{{ end }}

				<!-- LLRs -->
				<ul>
					{{ range .Parents }}
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
			{{ .Error }}
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
			{{ if .Matches $.Filter $.Diffs }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
			{{ range .Children }}
				{{ if .Matches $.Filter $.Diffs }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
				{{ range .Children }}
					{{ if .Matches $.Filter $.Diffs }}
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
		{{ if shouldShowTag . }}
			{{ range .Parents }}
				{{ if .Matches $.Filter $.Diffs }}
					{{ with ($.Once.Once .) }}
						{{ template "REQUIREMENT" . }}
						{{ template "CODETAGS" .Tags }}
						{{ template "CHANGELIST" .Changelists }}
					{{ end }}
				{{ end }}
				{{ range .Parents }}
					{{ if .Matches $.Filter $.Diffs }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
						{{ range .Parents }}
							{{ if .Matches $.Filter $.Diffs }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
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
			{{ .Error }}
		</li>
	{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}
`
