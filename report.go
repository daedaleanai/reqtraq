package main

import (
	"html/template"
	"io"
	"log"
	"os/exec"
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
				max-width: 1200px;
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
// @llr REQ-TRAQ-SWL-19
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

var reportTmpl = template.Must(template.Must(template.New("").Funcs(template.FuncMap{"formatBodyAsHTML": formatBodyAsHTML}).Parse(headerFooterTmplText)).Parse(reportTmplText))

var reportTmplText = `
{{ define "REQUIREMENT" }}
	{{if ne .Level -1 }}
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
		{{ template "STATUSFIELD" . }}
	{{ else }}
		<h3><a href="#{{ .ID }}">{{ .ID }} {{ .Title }}</a></h3>
 	{{end}}
{{ end }}

{{ define "CODETAGS"}}
	{{ if . }}
		<p>Code:
		{{ range . }}
			<a href="{{ .URL }}" target="_blank">{{ .Path }}:{{ .Tag }}</a>
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
		{{ range .Reqs.CodeFiles }}
		{{ with index $.Reqs.CodeTags . }}
		{{ range . }}
			<li>
				<h3><a href="{{ .URL }}" target="_blank">{{ .Path }}:{{ .Tag }}</a></h3>

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
	
	<h3>Basic</h3>
	<ul>
	{{ range .Reqs.Errors }}
		<li>
			{{ . }}
		</li>
	{{ else }}
		<li class="text-success">No basic errors found.</li>
	{{ end }}
	</ul>

	<h3>Invalid Attributes</h3>
	<ul>
	{{ range .AttributesErrors }}
		<li>
			{{ . }}
		</li>
	{{ else }}
		<li class="text-success">No attributes errors found.</li>
	{{ end }}
	</ul>

	<h3>Invalid References</h3>
	<ul>
	{{ range .ReferencesErrors }}
	  <li>
			{{ . }}
		</li>
	{{ else }}
		<li class="text-success">No references errors found.</li>
	{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}

{{ define "TOPDOWNFILT"}}
	{{template "HEADER"}}
	<h1>Top Down Tracing</h1>
	
	<h3><em>Filter Criteria: {{ $.Filter }} </em></h3>
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
	
	<h3><em>Filter Criteria: {{ $.Filter }} </em></h3>
	<ul style="list-style: none; padding: 0; margin: 0;">
		{{ range .Reqs.CodeFiles }}
		{{ with index $.Reqs.CodeTags . }}
		{{ range . }}
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

	<h3><em>Filter Criteria: {{ $.Filter }} </em></h3>
	<h3>Basic</h3>
	<ul>
	{{ range .Reqs.Errors }}
		<li>
			{{ . }}
		</li>
	{{ end }}
	</ul>
	<h3>Invalid Attributes</h3>
	<ul>
	{{ range .AttributesErrors }}
		<li>
			{{ . }}
		</li>
	{{ end }}
	</ul>
	<h3>Invalid References</h3>
	<ul>
	{{ range .ReferencesErrors }}
		<li>
			{{ . }}
		</li>
	{{ end }}
	</ul>
	{{ template "FOOTER" }}
{{ end }}
`

type reportData struct {
	Reqs             reqGraph
	Filter           *ReqFilter
	Once             Oncer
	Diffs            map[string][]string
	AttributesErrors []error
	ReferencesErrors []error
}

func (rg reqGraph) ReportDown(w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "TOPDOWN", reportData{rg, nil, Oncer{}, nil, nil, nil})
}

func (rg reqGraph) ReportUp(w io.Writer) error {
	return reportTmpl.ExecuteTemplate(w, "BOTTOMUP", reportData{rg, nil, Oncer{}, nil, nil, nil})
}

func (rg reqGraph) ReportIssues(w io.Writer) error {
	conf, err := parseConf(*fReportConfPath)
	if err != nil {
		return err
	}
	attributesErrors, err := rg.CheckAttributes(conf, nil, nil)
	if err != nil {
		return err
	}
	referencesErrors, err := rg.checkReqReferences(*fCertdocPath)
	if err != nil {
		return err
	}
	return reportTmpl.ExecuteTemplate(w, "ISSUES", reportData{rg, nil, Oncer{}, nil, attributesErrors, referencesErrors})
}

// @llr REQ-TRAQ-SWL-6
func (rg reqGraph) ReportDownFiltered(w io.Writer, f *ReqFilter, diffs map[string][]string) error {
	return reportTmpl.ExecuteTemplate(w, "TOPDOWNFILT", reportData{rg, f, Oncer{}, diffs, nil, nil})
}

// @llr REQ-TRAQ-SWL-7
func (rg reqGraph) ReportUpFiltered(w io.Writer, f *ReqFilter, diffs map[string][]string) error {
	return reportTmpl.ExecuteTemplate(w, "BOTTOMUPFILT", reportData{rg, f, Oncer{}, diffs, nil, nil})
}

func (rg reqGraph) ReportIssuesFiltered(w io.Writer, filter *ReqFilter, diffs map[string][]string) error {
	conf, err := parseConf(*fReportConfPath)
	if err != nil {
		return err
	}
	attributesErrors, err := rg.CheckAttributes(conf, filter, diffs)
	if err != nil {
		return err
	}
	// TODO(ab): Allow filtering references errors.
	referencesErrors, err := rg.checkReqReferences(*fCertdocPath)
	if err != nil {
		return err
	}
	return reportTmpl.ExecuteTemplate(w, "ISSUESFILT", reportData{rg, filter, Oncer{}, diffs, attributesErrors, referencesErrors})
}
