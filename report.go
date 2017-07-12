package main

import (
	"html/template"
	"io"
)

type Oncer map[string]bool

func (o Oncer) Once(r *Req) *Req {
	ok := o[r.ID]
	o[r.ID] = true
	if !ok {
		return r
	}
	return &Req{ID: r.ID, Title: r.Title, Body: r.Body, Level: -1}
}

var reportTmpl = template.Must(template.New("").Parse(`
{{ define "REQUIREMENT" }}
	{{if ne .Level -1 }}
		<h3><a name="{{ .ID }}"></a>{{ .ID }} {{ .Title }}</h3>
		{{ if .Body }}
			<p>{{ .Body }}</p>
		{{ end }}
		{{ if .Attributes }}
			<ul style="list-style: none; padding: 0; margin: 0;">
			{{ range $k, $v := .Attributes }}
				<li><strong>{{ $k }}</strong>: {{ $v }}</li>
			{{ end }}
			</ul>
		{{ end }}
		{{ template "STATUSFIELD" . }}
	{{ else }}
		<h3><a href="#{{ .ID }}">{{ .ID }} {{ .Title }}</a></h3>
 	{{end}}
{{ end }}

{{ define "CODEFILES"}}
	<p>Code Files:
		{{ range . }}
			<a href="file://{{ .Path }}" target="_blank">{{ .ID }}</a>
		{{ else }}
			<span class="text-danger">No code files</span>
		{{ end }}
	</p>
{{ end }}

{{ define "CHANGELIST" }}
	<p>Changelists:
		{{ range $k, $v := . }}
			<a href="{{ $v }}" target="_blank"><span class="label label-primary">{{ $k }}</span></a>
		{{ else }}
			<span class="text-danger">No changelist</span>
		{{ end }}
	</p>
{{ end }}

{{ define "STATUSFIELD" }}
	<p>Status:
		{{ if eq .Status 0 }}
			<span class="label label-default">{{ .Status }}</span>
		{{ else if eq .Status 1 }}
			<span class="label label-primary">{{ .Status }}</span>
		{{ else }}
			<span class="label label-success">{{ .Status }}</span>
		{{ end }}
{{ end }}

{{ define "PROBLEMREPORTS" }}
	<p>Problem Reports:
		{{ range $k, $v := . }}
			{{if $v.IsClosed}}
				<a href="{{ $v.URI }}" target="_blank"> <span class="label label-success">T{{ $v.DisplayID }}</span></a>
			{{else}}
				<a href="{{ $v.URI }}" target="_blank"> <span class="label label-danger">T{{ $v.DisplayID }}</span></a>
			{{end}}
		{{ else }}
			<span class="text-danger">No problem reports</span>
		{{ end }}
	</p>
{{ end }}

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
			body {
				font-family: Roboto, Arial, sans-serif;
				max-width: 1200px;
				margin-left: 5%;
				margin-right: 5%;
			}
			a, a:hover {
				text-decoration: none;
			}
		</style>
		<!-- Load MathJax for rendering of equations -->
		<script type="text/javascript" async
			src="https://cdnjs.cloudflare.com/ajax/libs/mathjax/2.7.1/MathJax.js?config=TeX-AMS-MML_HTMLorMML">
		</script>

	</head>
	<body>
		<section style="max-width:100%; text-align:center;">
			<h1>Reqtraq Report</h1>

{{end}}
{{define "FOOTER"}}
	</body>
</html>
{{end}}

{{define "TOPDOWN"}}
	{{template "HEADER"}}
		<h2>Top Down Tracing</h2>
		<hr>
	</section>
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
									{{ template "REQUIREMENT" ($.Once.Once .) }}
									{{ template "CODEFILES" .Children }}
									{{ template "CHANGELIST" .Changelists }}
									{{ template "PROBLEMREPORTS" .Tasklists }}
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
		<h2>Bottom Up Tracing</h2>
		<hr>
	</section>
	<ul style="list-style: none; padding: 0; margin: 0;">
		{{ range .Reqs.CodeFilesByPosition }}
			<li>
				<h3><a href="{{ .Path }}" target="_blank">{{ .ID }}</a></h3>
				{{ template "STATUSFIELD" . }}
				<!-- LLRs -->
				<ul>
				{{ range .Parents }}
					<li>
						{{ template "REQUIREMENT" ($.Once.Once .) }}
						{{ template "CHANGELIST" .Changelists }}
						{{ template "PROBLEMREPORTS" .Tasklists }}

						<!-- HLRs -->
							<ul>
							{{ range .Parents }}
								<li>
									{{ template "REQUIREMENT" ($.Once.Once .) }}
									<!-- SYSTEM -->
									<ul>
									{{ range .Parents }}
										<li>
											{{ template "REQUIREMENT" ($.Once.Once .) }}
										</li>
									{{ else }}
										<li class="text-danger">No parents</li>
									{{ end }}
									</ul>
								</li>
							{{ else }}
								<li class="text-danger">No parents</li>
							{{ end }}
							</ul>
					</li>
					{{ else }}
						<li class="text-danger">No parents</li>
					{{ end }}
				</ul>
			</li>
		{{ else }}
			<li class="text-danger">Empty graph</li>
		{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}


{{ define "ISSUES" }}
	{{template "HEADER"}}
		<h2>Issues</h2>
		<hr>
	</section>
	<h3>Dangling Requirements:</h3>
	<ul>
	{{ range .Reqs.DanglingReqsByPosition }}
		<li>
			{{ template "REQUIREMENT" ($.Once.Once .) }}
		</li>
	{{ else }}
		<li class="text-success">No dangling HLRs or LLRs found.</li>
	{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}

{{ define "TOPDOWNFILT"}}
	{{template "HEADER"}}
		<h2>Top Down Tracing</h2>
		<hr>
	</section>
	<h3><em>Filter Criteria: {{ $.Filter }} </em></h3>
	<ul style="list-style: none; padding: 0; margin: 0;">
		{{ range .Reqs.OrdsByPosition }}
			{{ if .Matches $.Filter $.Diffs }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
			{{ range .Children }}
				{{ if .Matches $.Filter $.Diffs }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
				{{ range .Children }}
					{{ if .Matches $.Filter $.Diffs }}
						{{ template "REQUIREMENT" ($.Once.Once .) }}
						{{ template "CODEFILES" .Children }}
						{{ template "CHANGELIST" .Changelists }}
						{{ template "PROBLEMREPORTS" .Tasklists }}

					{{ end }}
				{{ end }}
			{{ end }}
		{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}

{{ define "BOTTOMUPFILT" }}
	{{template "HEADER" }}
		<h2>Bottom Up Tracing</h2>
		<hr>
	</section>
	<h3><em>Filter Criteria: {{ $.Filter }} </em></h3>
	<ul style="list-style: none; padding: 0; margin: 0;">
		{{ range .Reqs.CodeFilesByPosition }}
			{{ range .Parents }}
				{{ if .Matches $.Filter $.Diffs }}
					{{ template "REQUIREMENT" ($.Once.Once .) }}
					{{ template "CODEFILES" .Children }}
					{{ template "CHANGELIST" .Changelists }}
					{{ template "PROBLEMREPORTS" .Tasklists }}

				{{ end }}
				{{ range .Parents }}
					{{ if .Matches $.Filter $.Diffs }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
						{{ range .Parents }}
							{{ if .Matches $.Filter $.Diffs }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
						{{ end }}
				{{ end }}
			{{ end }}
		{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}

{{ define "ISSUESFILT" }}
	{{template "HEADER"}}
		<h2>Issues</h2>
		<hr>
	</section>
	<h3><em>Filter Criteria: {{ $.Filter }} </em></h3>
	<h3>Dangling Requirements:</h3>
	<ul>
	{{ range .Reqs.DanglingReqsByPosition }}
		<li>
			{{ if .Matches $.Filter  }}{{ template "REQUIREMENT" ($.Once.Once .) }}{{ end }}
		</li>
	{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}
`))

type reportData struct {
	Reqs   reqGraph
	Filter ReqFilter
	Once   Oncer
	Diffs  map[string][]string
}

func (rg reqGraph) ReportDown(w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "TOPDOWN", reportData{rg, nil, Oncer{}, nil})
}

func (rg reqGraph) ReportUp(w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "BOTTOMUP", reportData{rg, nil, Oncer{}, nil})
}

func (rg reqGraph) ReportIssues(w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "ISSUES", reportData{rg, nil, Oncer{}, nil})
}

// @llr REQ-TRAQ-SWL-6
func (rg reqGraph) ReportDownFiltered(w io.Writer, f ReqFilter, diffs map[string][]string) error {
	return reportTmpl.ExecuteTemplate(w, "TOPDOWNFILT", reportData{rg, f, Oncer{}, diffs})
}

// @llr REQ-TRAQ-SWL-7
func (rg reqGraph) ReportUpFiltered(w io.Writer, f ReqFilter, diffs map[string][]string) error {
	return reportTmpl.ExecuteTemplate(w, "BOTTOMUPFILT", reportData{rg, f, Oncer{}, diffs})
}

func (rg reqGraph) ReportIssuesFiltered(w io.Writer, f ReqFilter, diffs map[string][]string) error {
	return reportTmpl.ExecuteTemplate(w, "ISSUESFILT", reportData{rg, f, Oncer{}, diffs})
}
