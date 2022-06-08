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
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
)

// Serve starts the web server listening on the supplied address:port
// @llr REQ-TRAQ-SWL-37
func Serve(addr string, reqtraqConfig *config.Config) error {
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

func Title(str string) string {
	return strings.Title(strings.ToLower(str))
}

var indexTemplate = template.Must(template.New("index").Funcs(template.FuncMap{"title": Title}).Parse(
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
<h1><img src="https://static.tildacdn.com/tild3132-3161-4531-b932-626532316433/favicon.ico"> {{.RepoName}}</h1>

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
{{ range $attrName, $attr := .Attributes }}
<div class="rTableRow">
<div class="rTableCell" style="padding-left: 2em;">{{ title $attrName }}:</div>
<div class="rTableCell"><input name="attribute_filter_{{ title $attrName }}" type="text"></div>
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
	{{ range $childReqSpec, $parentReqSpec := .ReqSpecLinks }}
		<div>
			<div>
				<a href="/matrix?from=REQ-{{ $parentReqSpec.Prefix }}-{{ $parentReqSpec.Level }}&to=REQ-{{ $childReqSpec.Prefix }}-{{ $childReqSpec.Level }}">
					REQ-{{ $parentReqSpec.Prefix }}-{{ $parentReqSpec.Level }} -> REQ-{{ $childReqSpec.Prefix }}-{{ $childReqSpec.Level }}
				</a>
			</div>
		</div>
	{{ end }}

	{{ range $reqSpec := .CodeLinks }}
		<div>
			<div>
				<a href="/matrix?from=REQ-{{ $reqSpec.Prefix }}-{{ $reqSpec.Level }}&to=CODE">
					REQ-{{ $reqSpec.Prefix }}-{{ $reqSpec.Level }} -> CODE
				</a>
			</div>
		</div>
	{{ end }}
	</div>
</div>
</body>
</html>`))

type indexData struct {
	RepoName     string
	Attributes   map[string]*config.Attribute
	Commits      []string
	ReqSpecLinks map[config.ReqSpec]config.ReqSpec
	CodeLinks    []config.ReqSpec
}

func parseReqSpecFromRequest(specString string) (config.ReqSpec, error) {
	if !strings.HasPrefix(specString, "REQ-") {
		return config.ReqSpec{}, fmt.Errorf("Invalid requirement specification `%s`", specString)
	}

	parts := strings.Split(strings.TrimPrefix(specString, "REQ-"), "-")
	if len(parts) != 2 {
		return config.ReqSpec{}, fmt.Errorf("Invalid requirement specification `%s`", specString)
	}
	return config.ReqSpec{
		Prefix: config.ReqPrefix(parts[0]),
		Level:  config.ReqLevel(parts[1]),
	}, nil
}

// get provides the page information for a given request
// @llr REQ-TRAQ-SWL-37
func get(w http.ResponseWriter, r *http.Request) error {
	repoName := repos.BaseRepoName()
	reqPath := r.URL.Path

	reqtraqConfig, err := config.ParseConfig(repos.BaseRepoPath())
	if err != nil {
		log.Fatal("Error parsing `reqtraq_config.json` file in current repo:", err)
	}

	// root page
	if reqPath == "/" {
		commits, err := repos.AllCommits(repoName)
		if err != nil {
			return err
		}
		attributes := make(map[string]*config.Attribute)
		codeLinks := []config.ReqSpec{}

		for _, repo := range reqtraqConfig.Repos {
			for _, document := range repo.Documents {
				for attributeName, attribute := range document.Schema.Attributes {
					if _, ok := attributes[attributeName]; !ok {
						attributes[attributeName] = attribute
					}
				}
				if document.HasImplementation() {
					codeLinks = append(codeLinks, document.ReqSpec)
				}
			}
		}
		reqSpecLinks := reqtraqConfig.GetLinkedReqSpecs()

		return indexTemplate.Execute(w, indexData{string(repoName), attributes, commits, reqSpecLinks, codeLinks})
	}

	// code files linked to from reports
	if strings.HasPrefix(reqPath, "/code/") {
		lexer := lexers.Match(reqPath)
		if lexer == nil {
			return errors.New("unknown file type")
		}

		path := strings.TrimPrefix(reqPath, "/code/")
		parts := strings.SplitN(path, "/", 2)
		repoName := repos.RepoName(parts[0])
		filePath := parts[1]

		filePath, err := repos.PathInRepo(repoName, filePath)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}
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
	rg, err := buildGraph(atCommit, &reqtraqConfig)
	if err != nil {
		return err
	}

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
			prg, err = buildGraph(sinceCommit, &reqtraqConfig)
			if err != nil {
				return err
			}
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
		fromSpec, err := parseReqSpecFromRequest(r.FormValue("from"))
		if err != nil {
			return err
		}

		to := r.FormValue("to")
		if to == "CODE" {
			return rg.GenerateCodeTraceTables(w, fromSpec)
		}

		toSpec, err := parseReqSpecFromRequest(to)
		if err != nil {
			return err
		}
		return rg.GenerateTraceTables(w, fromSpec, toSpec)
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
