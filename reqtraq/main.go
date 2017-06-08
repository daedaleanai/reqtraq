/*
 * ReqTraq is the swiss army knife binary implementing all requirements tracking and linting for prod repo's at Daedalean.
 * Run without arguments to get comprehensive help.
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"github.com/daedaleanai/reqtraq/git"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/daedaleanai/reqtraq/lyx"
	"strings"
)

var (
	fReportPrefix            = flag.String("pfx", "./req-", "path and filename prefix for reports.")
	fReportTitleFilterString = flag.String("title_filter", "", "regular expression to filter by requirement title.")
	fReportIdFilterString    = flag.String("id_filter", "", "regular expression to filter by requirement id.")
	fReportBodyFilterString  = flag.String("body_filter", "", "regular expression to filter by requirement body.")
	fReportJsonConfPath      = flag.String("attributes", git.RepoPath()+"/certdocs/attributes.json", "path to json with requirement attribute specification.")
	addr                     = flag.String("addr", ":8080", "The ip:port where to serve.")
	since                    = flag.String("since", "", "The commit representing the start of the range.")
	at                       = flag.String("at", "", "The commit representing the end of the range.")
	fCertdocPath             = flag.String("certdoc_path", "certdocs", "Location of certification documents within the *root* of the current repository.")
	fCodePath                = flag.String("code_path", "", "Location of code files within the current repository")
	fVerbose                 = flag.Bool("v", false, "Enable verbose logs.")
)

const usage = `
Syntax:

	reqtraq command <command_args> [flags]

ReqTraq is a requirements tracer.

ReqTraq operates on .lyx documents and source code in a directory tree, usually
in a git repo.  The .lyx documents are scanned for requirements, and the source code
for tags referring to them.

command is one of:
	help		prints this help message
	linkify		changes the lyx content by adding named destinations and links to parent requirements
	list    	parses and lists all requirements found in .lyx files
	nextid		generates the next requirement id for the given document
	precommit	runs the precommit checks for the requirement documents in the current repository
	prepush		runs the prepush checks for the requirement documents in the current repository
	reportdown 	creates an HTML traceability report from system requirements down to code
	reportissues	creates an HTML report with all issues found in the requirement documents
	reportup 	creates an HTML traceability report from code, to LLRs, to HLRs and to system requirements
	updatetasks	updates the tasks associated with the given requirements (requires a Phabricator/JIRA/Bugzilla instance)
	web		starts a local web server to facilitate interaction with reqtraq



Invoking reqtraq without arguments prints a short help message.
Run
	reqtraq help <command>
for more information on a specific command`

const linkifyUsage = `Changes the lyx content by adding named destinations and links to parent requirements. Usage:
	reqtraq linkify <input_lyx_filename> <output_lyx_filename>
Parameters:
	<input_lyx_filename>	Lyx file to be linkified
	<output_lyx_filename>	linkified Lyx file
`

const listUsage = `Parses and lists all requirements found in .lyx files. Usage:
	reqtraq linkify <input_lyx_filename>
Parameters:
	<input_lyx_filename>	Lyx file to be parsed
`

const nextidUsage = `Generates the next requirement id for the given document. Usage:
	reqtraq nextid <input_lyx_filename>
Parameters:
	<input_lyx_filename>	Lyx file to generate the next requirement id for
`

const precommitUsage = `Runs the pre-commit checks for the requirement documents in the current repository. Usage:
	reqtraq precommit --certdoc_path=<path>
Parameters:
	--certdoc_path: location of certification documents within the current repository

If the binary exits with a 0 exitcode, the requirement documents are correct. A non-zero exit code signals one or more
problems, which are printed to stderr.
`

const prepushUsage = `Runs the pre-push checks for the requirement documents in the current repository. Usage:
	reqtraq prepush --certdoc_path=<path>
Parameters:
	--certdoc_path: location of certification documents within the current repository

If the binary exits with a 0 exitcode, the pre-push ran successfully. A non-zero exit code signals one or more
problems, which are printed to stderr.
`

const reportUsage = `
	reportdown 	creates an HTML traceability report from system requirements down to code
	reportissues	creates an HTML report with all issues found in the requirement documents
	reportup 	creates an HTML traceability report from code, to LLRs, to HLRs and to system requirements
Usage:
	reqtraq report<type> --pfx=<reportfile-prefix> --title_filter=<regexp> --id_filter=<regexp>
		--body_filter=<regexp> --attributes=<path_to_attributes_json> --since=<start_commid> --at=<end_commit>
		--certdoc_path=<path>
Parameters:
	--pfx: path and filename prefix for reports.
	--title_filter: regular expression to filter by requirement title.
	--id_filter: regular expression to filter by requirement id.
	--body_filter: regular expression to filter by requirement body.
	--attributes: path to json with requirement attribute specification.
	--since: the Git commit SHA-1 representing the start of the range.
	--at: the commit representing the end of the range.
	--certdoc_path: location of certification documents within the current repository
`

const updateTaskUsage = `Updates the tasks associated with the given requirements (requires a Phabricator/JIRA/Bugzilla instance). Usage:
	reqtraq updatetasks --certdoc_path=<path>
Parameters:
	--certdoc_path: location of certification documents within the current repository

For each requirement the method will:
	- find the task associated with the requirement, by searching for the requirement ID in the task title using the taskmgr API
	- if a task was found and the requirement was not deleted, its title and description are updated
	- if a task was found and the requirement was deleted, the task is set as INVALID
	- if the task was not found, it is created and filled in with the following values:
	 	Title: <Req ID> <Req Title>
		Description: <Requirement Body>
		Status: Open
		Tags: Project Abbreviation (e.g. DDLN, VXU, etc.)
      		Parents: the first parent task (Phabricator doesn't yet support multiple parents in the api)
`

const webUsage = `Starts a local web server to facilitate interaction with reqtraq. Usage:
	reqtraq web --addr="hostport" --certdoc_path=<path>
Parameters:
	--addr: the ip:port where to serve.
	--certdoc_path: location of certification documents within the current repository.

`

type JsonConf struct {
	Attributes []map[string]string
}

func showHelp() {
	subCommand := ""
	if len(os.Args) > 1 {
		subCommand = os.Args[1]
	}
	switch subCommand {
	case "help", "": // general help
		fmt.Println(usage)
	case "linkify":
		fmt.Println(linkifyUsage)
	case "list":
		fmt.Println(listUsage)
	case "nextid":
		fmt.Println(nextidUsage)
	case "precommit":
		fmt.Println(precommitUsage)
	case "prepush":
		fmt.Println(prepushUsage)
	case "reportup", "reportdown", "reportissues":
		fmt.Println(reportUsage)
	case "updatetasks":
		fmt.Println(updateTaskUsage)
	case "web":
		fmt.Println(webUsage)
	default:
		fmt.Printf("Unknown command '%s'", subCommand)
		fmt.Println(usage)
	}
}

func main() {
	flag.Parse()
	command := flag.Arg(0)
	if command == "" {
		command = "help"
	}

	var err error

	linepipes.Verbose = *fVerbose

	// check to see if the command has a second parameter, e.g. list <filename>
	f := ""
	remainingArgs := flag.Args()
	if len(remainingArgs) > 1 {
		if !strings.HasPrefix(remainingArgs[1], "-") {
			f = remainingArgs[1]
		}
		// See maybe there are more flags after the `action`.
		os.Args = append(os.Args[:1], remainingArgs[1:]...)
		flag.Parse()
	}

	filter := ReqFilter{} // Filter for report generation
	switch command {
	case "reportdown", "reportup", "reportissues":
		if len(*fReportTitleFilterString) > 0 {
			filter[TitleFilter], err = regexp.Compile(*fReportTitleFilterString)
			if err != nil {
				log.Fatal(err)
			}
		}
		if len(*fReportIdFilterString) > 0 {
			filter[IdFilter], err = regexp.Compile(*fReportIdFilterString)
			if err != nil {
				log.Fatal(err)
			}
		}
		if len(*fReportBodyFilterString) > 0 {
			filter[BodyFilter], err = regexp.Compile(*fReportBodyFilterString)
			if err != nil {
				log.Fatal(err)
			}
		}
	case "help":
		showHelp()
		os.Exit(0)
	case "linkify", "list", "nextid":
		if f == "" {
			log.Fatal("Missing file name")
		}
	}

	var (
		rg, prg reqGraph
		diffs   map[string][]string
	)
	switch command {
	case "reportdown", "reportup", "reportissues", "prepush":
		var dir string
		rg, dir, err = buildGraph(*at)
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dir)

		if *since != "" {
			var dir string
			prg, dir, err = buildGraph(*since)
			if err != nil {
				log.Println(err)
			}
			defer os.RemoveAll(dir)
		}
		diffs = rg.ChangedSince(prg)
	}

	switch command {
	case "nextid":
		nextID, err := lyx.NextId(f)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(nextID)
	case "list":
		reqs, err := lyx.ParseCertdoc(f, ioutil.Discard)
		if err != nil {
			log.Fatal(err)
		}
		failureCount := 0
		for _, v := range reqs {
			r, err2 := lyx.ParseReq(v)
			body := strings.Split(r.Attributes["TEXT"], "\n")
			fmt.Printf("Requirement %s %s\n%s...\n\n", r.ID, body[0], body[1])
			if err2 != nil {
				failureCount++
			}
		}
		if failureCount > 0 {
			log.Fatalf("Requirements failed to parse: %d", failureCount)
		}
	case "linkify":
		output := flag.Arg(1)
		if output == "" {
			log.Fatal("Missing output file name")
		}
		o, err := os.Create(output)
		if err != nil {
			log.Fatal(err)
		}
		_, err = lyx.ParseCertdoc(f, o)

		if err != nil {
			log.Fatal(err)
		}
	case "reportdown":
		of, err := os.Create(*fReportPrefix + "down.html")
		if err != nil {
			log.Fatal(err)
		}
		logFileCreate(of.Name())
		if err := rg.ReportDown(of); err != nil {
			log.Fatal(err)
		}
		of.Close()

		if len(filter) > 0 || diffs != nil {
			of, err := os.Create(*fReportPrefix + "down-filtered.html")
			if err != nil {
				log.Fatal(err)
			}
			logFileCreate(of.Name())
			if err := rg.ReportDownFiltered(of, filter, diffs); err != nil {
				log.Fatal(err)
			}
			of.Close()
		}
	case "reportup":
		of, err := os.Create(*fReportPrefix + "up.html")
		if err != nil {
			log.Fatal(err)
		}
		logFileCreate(of.Name())
		if err = rg.ReportUp(of); err != nil {
			log.Fatal(err)
		}
		of.Close()

		if len(filter) > 0 || diffs != nil {
			of, err := os.Create(*fReportPrefix + "up-filtered.html")
			if err != nil {
				log.Fatal(err)
			}
			logFileCreate(of.Name())
			if err := rg.ReportUpFiltered(of, filter, diffs); err != nil {
				log.Fatal(err)
			}
			of.Close()
		}
	case "reportissues":
		of, err := os.Create(*fReportPrefix + "issues.html")
		if err != nil {
			log.Fatal(err)
		}
		logFileCreate(of.Name())
		if err := rg.ReportIssues(of); err != nil {
			log.Fatal(err)
		}
		of.Close()
		if len(filter) > 0 || diffs != nil {
			of, err := os.Create(*fReportPrefix + "issues-filtered.html")
			if err != nil {
				log.Fatal(err)
			}
			logFileCreate(of.Name())
			if err := rg.ReportIssuesFiltered(of, filter, diffs); err != nil {
				log.Fatal(err)
			}
			of.Close()
		}
	case "web":
		err := serve(*addr)
		if err != nil {
			log.Fatal(err)
		}
	case "precommit":
		err := precommit(*fCertdocPath, *fCodePath, *fReportJsonConfPath)
		if err != nil {
			log.Fatal(err)
		}
	case "prepush":
		changedReqIds := map[string]bool{}
		for k := range diffs {
			changedReqIds[k] = true
			fmt.Println("Changed requirement ", k)
		}
		if err := rg.UpdateTasks(changedReqIds); err != nil {
			log.Fatal(err)
		}
	case "updatetasks": // update all task title/descriptions/attributes based on the requirement documents
		rg, err := CreateReqGraph(*fCertdocPath, *fCodePath)
		if err != nil {
			log.Fatal(err)
		}
		reqIds := map[string]bool{}
		for k := range rg {
			reqIds[k] = true
		}
		if err := rg.UpdateTasks(reqIds); err != nil {
			log.Fatal(err)
		}
	}
}

func logFileCreate(fileName string) {
	log.Print("Creating ", fileName, " (this may take a while)...")
}

func precommit(certdocPath, codePath, reportJsonConfPath string) error {
	var reportConf JsonConf
	b, err := ioutil.ReadFile(reportJsonConfPath)
	if err != nil {
		fmt.Printf("Can't find attributes.json in '%s'. Attributes won't be checked.\n",
			reportJsonConfPath)
		reportConf = JsonConf{}
	} else {
		if err := json.Unmarshal(b, &reportConf); err != nil {
			return fmt.Errorf("Error while parsing attributes: ", err)
		}
	}

	rg, err := CreateReqGraph(certdocPath, codePath)
	if err != nil {
		return err
	}
	errorResult := ""
	err = rg.checkReqReferences(certdocPath)
	if err != nil {
		errorResult += err.Error()
	}

	if errs := rg.CheckAttributes(reportConf.Attributes); len(errs) > 0 {
		for _, e := range errs {
			errorResult += e.Error()
		}
	}
	if errorResult == "" {
		return nil
	} else {
		return fmt.Errorf(errorResult)
	}
}

func buildGraph(commit string) (reqGraph, string, error) {
	if commit == "" {
		rg, err := CreateReqGraph(*fCertdocPath, *fCodePath)
		return rg, "", err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	dir, err := git.Clone()
	if err != nil {
		return nil, dir, err
	}
	if err = git.Checkout(commit); err != nil {
		return nil, dir, err
	}
	rg, err := CreateReqGraph(*fCertdocPath, *fCodePath)
	if err != nil {
		return nil, dir, err
	}
	if err := os.Chdir(cwd); err != nil {
		return nil, dir, err
	}
	return rg, dir, nil
}
