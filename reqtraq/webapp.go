// @llr REQ-0-DDLN-SWL-016
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/daedaleanai/reqtraq/git"
)

func serve(addr string) error {
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}
	fmt.Printf("Server started on http://%s\n", addr)
	return http.ListenAndServe(addr, http.HandlerFunc(handler))
}

var errorTemplate *template.Template = template.Must(template.New("error").Parse(
	`<html>OOPS, {{.Error}}`))

func handler(w http.ResponseWriter, r *http.Request) {
	log.Print(r.Method, r.URL)
	var err error
	switch r.Method {
	case "GET":
		err = get(w, r)
	default:
		err = fmt.Errorf("Unknown HTTP method: %s", r.Method)
	}
	if err != nil {
		_ = errorTemplate.Execute(w, err)
	}
}

var indexTemplate *template.Template = template.Must(template.New("index").Parse(
	`<!DOCTYPE html>
<html lang="en">
<head>
<title>{{.RepoName}}</title>
<style>
.rTable {
  	display: table;
}
.rTableRow {
  	display: table-row;
}
.rTableCell {
  	display: table-cell;
  	padding: 3px 10px;
}
.rTableBody {
  	display: table-row-group;
}
</style>
</head>

<body>
<h1><img src="https://www.daedalean.ai/favicon-32x32.png"> {{.RepoName}}</h1>

<form action="/report" method="get">
<p>Filter by:
<div class="rTable">
<div class="rTableRow">
<div class="rTableCell">ID:</div>
<div class="rTableCell"><input name="id_filter" type="text"></div>
</div>
<div class="rTableRow">
<div class="rTableCell">Title:</div>
<div class="rTableCell"><input name="title_filter" type="text"></div>
</div>
<div class="rTableRow">
<div class="rTableCell">Body:</div>
<div class="rTableCell"><input name="body_filter" type="text"></div>
</div>
<div class="rTableRow">
<div class="rTableCell">Since:</div>
<div class="rTableCell"><select name="since_commit">
<option value="">Beginning</option>
{{ range .Commits }}<option value="{{ . }}">{{ . }}</option>{{ end }}</select></div>
</div>
<div class="rTableRow">
<div class="rTableCell">At:</div>
<div class="rTableCell"><select name="at_commit">
<option value="">Current</option>
{{ range .Commits }}<option value="{{ . }}">{{ . }}</option>{{ end }}</select></div>
</div>
<div class="rTableRow">
<div class="rTableCell"></div>
<div class="rTableCell"><input type="reset"></div>
</div>
</div>
<input type="submit" name="report-type" value="Bottom Up"/>
<input type="submit" name="report-type" value="Top Down"/>
<input type="submit" name="report-type" value="Issues"/>
</p>
</form>
</body>
</html>`))

type indexData struct {
	RepoName string
	Commits  []string
}

func get(w http.ResponseWriter, r *http.Request) error {
	repoName := git.RepoName()
	path := r.URL.Path
	switch {
	case path == "/":
		commits, err := git.AllCommits()
		if err != nil {
			return err
		}
		indexTemplate.Execute(w, indexData{repoName, commits})

	case path == "/report":
		at := r.FormValue("at_commit")
		var atCommit string
		if at != "" {
			atCommit = strings.Split(at, " ")[0]
		}
		rg, dir, err := buildGraph(atCommit)
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
		filter := ReqFilter{}
		if len(r.FormValue("title_filter")) > 0 {
			filter[TitleFilter], err = regexp.Compile(r.FormValue("title_filter"))
			if err != nil {
				return err
			}
		}
		if len(r.FormValue("id_filter")) > 0 {
			filter[IdFilter], err = regexp.Compile(r.FormValue("id_filter"))
			if err != nil {
				return err
			}
		}
		if len(r.FormValue("body_filter")) > 0 {
			filter[BodyFilter], err = regexp.Compile(r.FormValue("body_filter"))
			if err != nil {
				return err
			}
		}
		var prg reqGraph
		since := r.FormValue("since_commit")
		if since != "" {
			sinceCommit := strings.Split(since, " ")[0]
			prg, dir, err = buildGraph(sinceCommit)
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
		}
		diffs := rg.ChangedSince(prg)
		switch r.FormValue("report-type") {
		case "Bottom Up":
			if len(filter) > 0 || diffs != nil {
				return rg.ReportUpFiltered(w, filter, diffs)
			}
			return rg.ReportUp(w)
		case "Top Down":
			if len(filter) > 0 || diffs != nil {
				return rg.ReportDownFiltered(w, filter, diffs)
			}
			return rg.ReportDown(w)
		case "Issues":
			if len(filter) > 0 || diffs != nil {
				return rg.ReportIssuesFiltered(w, filter, diffs)
			}
			return rg.ReportIssues(w)
		}
	}
	return nil
}
