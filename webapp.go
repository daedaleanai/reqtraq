/*
   Functions for creating and servicing a web interface.
*/
package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/daedaleanai/reqtraq/git"
	"github.com/pkg/errors"
)

// Serve starts the web server listening on the supplied address:port
// @llr REQ-TRAQ-SWL-37
func Serve(addr string) error {
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}
	fmt.Printf("Server started on http://%s\n", addr)
	return http.ListenAndServe(addr, http.HandlerFunc(handler))
}

var errorTemplate = template.Must(template.New("error").Parse(
	`<html>OOPS!
<pre>{{.Error}}</pre>`))

// handler responds to requests on the web server
// @llr REQ-TRAQ-SWL-37
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

var indexTemplate = template.Must(template.New("index").Parse(
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

.matrices {
	display: table;
	margin-right: 4em;
}
.matrices div {
	display: table-row;
}
.matrices div div {
	display: table-cell;
}
</style>
</head>

<body>
<h1><img src="https://www.daedalean.ai/favicon-32x32.png"> {{.RepoName}}</h1>

<h2>Reports</h2>
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
<div class="rTableCell">Attributes:</div>
<div class="rTableCell"><input name="any_attribute_filter" type="text"></div>
</div>
{{ range .Attributes }}
<div class="rTableRow">
<div class="rTableCell" style="padding-left: 2em;">{{ . }}:</div>
<div class="rTableCell"><input name="attribute_filter_{{ . }}" type="text"></div>
</div>
{{ end }}
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

<h2>Trace Matrices</h2>
<div style="display: flex;">
	<div class="matrices">
		<div>
			<div><a href="/matrix?from=SYS&to=SWH">SYS&ndash;SWH</a></div>
		</div>
		<div>
			<div><a href="/matrix?from=SWH&to=SWL">SWH&ndash;SWL</a></div>
		</div>
		<div>
			<div><a href="/matrix?from=SWL&to=CODE">SWL&ndash;CODE</a></div>
		</div>
	</div>

	<div class="matrices">
		<div>
			<div><a href="/matrix?from=SWH&to=HTC">SWH&ndash;HTC</a></div>
		</div>
		<div>
			<div><a href="/matrix?from=SWL&to=LTC">SWL&ndash;LTC</a></div>
		</div>
	</div>
</div>
</body>
</html>`))

type indexData struct {
	RepoName   string
	Attributes []string
	Commits    []string
}

// get provides the page information for a given request
// @llr REQ-TRAQ-SWL-37
func get(w http.ResponseWriter, r *http.Request) error {
	repoName := git.RepoName()
	reqPath := r.URL.Path

	// root page
	if reqPath == "/" {
		schema, err := ParseSchema(*fSchemaPath)
		if err != nil {
			return errors.Wrap(err, "Failed to parse config")
		}
		attributes := make([]string, 0, len(schema.Attributes)+1)
		for _, a := range schema.Attributes {
			attributes = append(attributes, a["name"])
		}
		commits, err := git.AllCommits()
		if err != nil {
			return err
		}
		return indexTemplate.Execute(w, indexData{repoName, attributes, commits})
	}

	// code files linked to from reports
	if strings.HasPrefix(reqPath, "/code/") {
		lexer := lexers.Match(reqPath)
		if lexer == nil {
			return errors.New("unknown file type")
		}
		filePath := path.Join(git.RepoPath(), reqPath[6:])
		contents, err := ioutil.ReadFile(filePath)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}
		iterator, err := lexer.Tokenise(nil, string(contents))
		formatter := html.New(html.Standalone(true), html.WithLineNumbers(true), html.LinkableLineNumbers(true, "L"), html.WithClasses(true))
		style := styles.Get("vs")
		return formatter.Format(w, style, iterator)
	}

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

	switch {
	case reqPath == "/report":
		filter, err := createFilterFromHttpRequest(r)
		if err != nil {
			return fmt.Errorf("Failed to create filter: %v", err)
		}
		var prg *ReqGraph
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
			if !filter.IsEmpty() || diffs != nil {
				return rg.ReportUpFiltered(w, filter, diffs)
			}
			return rg.ReportUp(w)
		case "Top Down":
			if !filter.IsEmpty() || diffs != nil {
				return rg.ReportDownFiltered(w, filter, diffs)
			}
			return rg.ReportDown(w)
		case "Issues":
			if !filter.IsEmpty() || diffs != nil {
				return rg.ReportIssuesFiltered(w, filter, diffs)
			}
			return rg.ReportIssues(w)
		}
	case reqPath == "/matrix":
		from := r.FormValue("from")
		to := r.FormValue("to")
		return rg.GenerateTraceTables(w, from, to)
	}
	return nil
}

// createFilterFromHttpRequest generates an appropriate report filter based on the web page form values
// @llr REQ-TRAQ-SWL-37
func createFilterFromHttpRequest(r *http.Request) (*ReqFilter, error) {
	filter := &ReqFilter{}
	filter.AttributeRegexp = make(map[string]*regexp.Regexp, 0)
	var err error
	if r.FormValue("id_filter") != "" {
		filter.IDRegexp, err = regexp.Compile(r.FormValue("id_filter"))
		if err != nil {
			return nil, errors.Wrap(err, "id_filter regex invalid")
		}
	}
	if r.FormValue("title_filter") != "" {
		filter.TitleRegexp, err = regexp.Compile(r.FormValue("title_filter"))
		if err != nil {
			return nil, errors.Wrap(err, "title_filter regex invalid")
		}
	}
	if r.FormValue("body_filter") != "" {
		filter.BodyRegexp, err = regexp.Compile(r.FormValue("body_filter"))
		if err != nil {
			return nil, errors.Wrap(err, "body_filter regex invalid")
		}
	}
	if r.FormValue("any_attribute_filter") != "" {
		filter.AnyAttributeRegexp, err = regexp.Compile(r.FormValue("any_attribute_filter"))
		if err != nil {
			return nil, errors.Wrap(err, "attributes regex invalid")
		}
	}
	for field := range r.Form {
		if !strings.HasPrefix(field, "attribute_filter_") {
			continue
		}
		attribute := strings.ToUpper(field[17:])
		rawValue := r.FormValue(field)
		if rawValue == "" {
			continue
		}
		value, err := regexp.Compile(rawValue)
		if err != nil {
			return nil, errors.Wrap(err, "title_filter regex invalid")
		}
		filter.AttributeRegexp[attribute] = value
	}
	return filter, nil
}
