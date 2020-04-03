/*
 * Reqtraq is the swiss army knife binary implementing all requirements tracking and linting for prod repo's at Daedalean.
 * Run without arguments to get comprehensive help.
 */

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/daedaleanai/reqtraq/git"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/pkg/errors"
)

var (
	fReportPrefix            = flag.String("pfx", "./req-", "path and filename prefix for reports.")
	fReportTitleFilterString = flag.String("title_filter", "", "regular expression to filter by requirement title.")
	fReportIdFilterString    = flag.String("id_filter", "", "regular expression to filter by requirement id.")
	fReportBodyFilterString  = flag.String("body_filter", "", "regular expression to filter by requirement body.")
	fReportConfPath          = flag.String("attributes", git.RepoPath()+"/certdocs/attributes.json", "path to json with requirement attribute specification.")
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

Reqtraq is a requirements tracer.

Reqtraq operates on certification documents and source code in a directory tree,
usually in a git repo.  The certification documents are scanned for requirements,
and the source code for references to them.

command is one of:
	help		prints this help message
	list    	parses and lists the requirements found in certification documents
	nextid		generates the next requirement id for the given document
	precommit	runs the precommit checks for the requirement documents in the current repository
	prepush		runs the prepush checks for the requirement documents in the current repository
	reportdown 	creates an HTML traceability report from system requirements down to code
	reportissues	creates an HTML report with all issues found in the requirement documents
	reportup 	creates an HTML traceability report from code, to LLRs, to HLRs and to system requirements
	web		starts a local web server to facilitate interaction with reqtraq



Invoking reqtraq without arguments prints a short help message.
Run
	reqtraq help <command>
for more information on a specific command`

const listUsage = `Parses and lists all requirements found in certification documents. Usage:
	reqtraq list <input_md_filename>
Parameters:
	<input_md_filename>	Markdown file to be parsed
`

const nextidUsage = `Generates the next requirement id for the given document. Usage:
	reqtraq nextid <input_md_filename>
Parameters:
	<input_md_filename>	Markdown file to generate the next requirement id for
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

	// assign global Verbose variable after arguments have been parsed
	linepipes.Verbose = *fVerbose

	filter := ReqFilter{} // Filter for report generation
	switch command {
	case "reportdown", "reportup", "reportissues":
		if len(*fReportTitleFilterString) > 0 {
			filter.TitleRegexp, err = regexp.Compile(*fReportTitleFilterString)
			if err != nil {
				log.Fatal(err)
			}
		}
		if len(*fReportIdFilterString) > 0 {
			filter.IDRegexp, err = regexp.Compile(*fReportIdFilterString)
			if err != nil {
				log.Fatal(err)
			}
		}
		if len(*fReportBodyFilterString) > 0 {
			filter.BodyRegexp, err = regexp.Compile(*fReportBodyFilterString)
			if err != nil {
				log.Fatal(err)
			}
		}
	case "help":
		showHelp()
		os.Exit(0)
	case "list", "nextid":
		if f == "" {
			log.Fatal("Missing file name")
		}
	}

	var (
		rg    *reqGraph
		diffs map[string][]string
	)
	switch command {
	case "reportdown", "reportup", "reportissues", "prepush":
		var dir string
		rg, dir, err = buildGraph(*at)
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dir)
	}
	switch command {
	case "reportdown", "reportup", "reportissues":
		var prg *reqGraph
		if *since != "" {
			var dir string
			prg, dir, err = buildGraph(*since)
			if err != nil {
				log.Fatal(err)
			}
			defer os.RemoveAll(dir)
		}
		diffs = rg.ChangedSince(prg)
	}

	switch command {
	case "nextid":
		nextID, err := NextId(f)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(nextID)
	case "list":
		reqs, err := ParseCertdoc(f)
		if err != nil {
			log.Fatal(err)
		}
		failureCount := 0
		for _, v := range reqs {
			r, err2 := ParseReq(v)
			if err2 != nil {
				log.Printf("Requirement failed to parse: %q\n%s", err2, v)
				failureCount++
				continue
			}
			body := make([]string, 0)
			lines := strings.Split(string(r.Body), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				body = append(body, line)
			}
			fmt.Printf("Requirement %s %s\n", r.ID, r.Title)
			// Check for empty body because deleted requirements have no body.
			if len(body) > 0 {
				fmt.Printf("%s…\n", body[0])
			}
			fmt.Println()
		}
		if failureCount > 0 {
			log.Fatalf("Requirements failed to parse: %d", failureCount)
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

		if !filter.IsEmpty() || diffs != nil {
			of, err := os.Create(*fReportPrefix + "down-filtered.html")
			if err != nil {
				log.Fatal(err)
			}
			logFileCreate(of.Name())
			if err := rg.ReportDownFiltered(of, &filter, diffs); err != nil {
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

		if !filter.IsEmpty() || diffs != nil {
			of, err := os.Create(*fReportPrefix + "up-filtered.html")
			if err != nil {
				log.Fatal(err)
			}
			logFileCreate(of.Name())
			if err := rg.ReportUpFiltered(of, &filter, diffs); err != nil {
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
		if !filter.IsEmpty() || diffs != nil {
			of, err := os.Create(*fReportPrefix + "issues-filtered.html")
			if err != nil {
				log.Fatal(err)
			}
			logFileCreate(of.Name())
			if err := rg.ReportIssuesFiltered(of, &filter, diffs); err != nil {
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
		err := precommit(*fCertdocPath, *fCodePath, *fReportConfPath)
		if err != nil {
			log.Fatal(err)
		}
	case "prepush":
		// Noop.
	default:
		fmt.Printf(`Invalid command "%s"`, command)
		fmt.Println("")
		fmt.Println(usage)
		os.Exit(1)
	}
}

func logFileCreate(fileName string) {
	log.Print("Creating ", fileName, " (this may take a while)...")
}

func parseConf(confPath string) (JsonConf, error) {
	var conf JsonConf
	b, err := ioutil.ReadFile(confPath)
	if err != nil {
		return conf, fmt.Errorf("Attributes specification file missing: %s", confPath)
	}
	if err := json.Unmarshal(b, &conf); err != nil {
		return conf, fmt.Errorf("Error while parsing attributes: %s", err)
	}
	return conf, nil
}

func precommit(certdocPath, codePath, confPath string) error {
	rg, err := CreateReqGraph(certdocPath, codePath)
	if err != nil {
		return err
	}
	errs := rg.Errors
	if len(errs) == 0 {
		errs2, err := rg.checkReqReferences(certdocPath)
		if err != nil {
			return err
		}
		errs = append(errs, errs2...)

		conf, err := parseConf(confPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("Failed to parse project config file: %s", err.Error()))
		} else {
			errs2, err = rg.CheckAttributes(conf, nil, nil)
			if err != nil {
				return err
			}
			errs = append(errs, errs2...)
		}
	}

	if len(errs) > 0 {
		return combineErrors(errs)
	}
	return nil
}

// combineErrors creates an error out of other errors.
func combineErrors(errs []error) error {
	var res bytes.Buffer
	for _, e := range errs {
		if res.Len() > 0 {
			res.WriteByte('\n')
		}
		res.WriteString(e.Error())
	}
	return errors.New(res.String())
}

// buildGraph returns the requirements graph at the specified commit, or
// the graph for the current files if commit is empty. In case the commit
// is specified, a temporary clone of the repository is created and the
// path to it is returned.
func buildGraph(commit string) (*reqGraph, string, error) {
	if commit == "" {
		rg, err := CreateReqGraph(*fCertdocPath, *fCodePath)
		return rg, "", errors.Wrap(err, "Failed to create graph in current dir")
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
		return rg, dir, errors.Wrap(err, fmt.Sprintf("Failed to create graph in %s", dir))
	}
	if err := os.Chdir(cwd); err != nil {
		return rg, dir, err
	}
	return rg, dir, nil
}
