/*
 * Reqtraq is the swiss army knife binary implementing all requirements tracking and linting for prod repo's at Daedalean.
 * Run without arguments to get comprehensive help.
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/git"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/pkg/errors"
)

var (
	fReportPrefix            = flag.String("pfx", "./req-", "path and filename prefix for reports.")
	fReportTitleFilterString = flag.String("title_filter", "", "regular expression to filter by requirement title.")
	fReportIdFilterString    = flag.String("id_filter", "", "regular expression to filter by requirement id.")
	fReportBodyFilterString  = flag.String("body_filter", "", "regular expression to filter by requirement body.")
	fAddr                    = flag.String("addr", ":8080", "The ip:port where to serve.")
	fSince                   = flag.String("since", "", "The commit representing the start of the range.")
	fAt                      = flag.String("at", "", "The commit representing the end of the range.")
	fCertdocPath             = flag.String("certdoc_path", "certdocs", "Location of certification documents within the *root* of the current repository.")
	fCodePath                = flag.String("code_path", "", "Location of code files within the current repository")
	fSchemaPath              = flag.String("schema_path", git.RepoPath()+"/certdocs/attributes.json", "path to json with requirement schema.")
	fStrict                  = flag.Bool("strict", false, "Exit with error if any validation checks fail.")
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
	reportdown 	creates an HTML traceability report from system requirements down to code
	reportissues	creates an HTML report with all issues found in the requirement documents
	reportup 	creates an HTML traceability report from code, to LLRs, to HLRs and to system requirements
	validate	validates the requirement documents in the current repository
	web		starts a local web server to facilitate interaction with reqtraq



Invoking reqtraq without arguments prints a short help message.
Run
	reqtraq help <command>
for more information on a specific command`

const listUsage = `Parses and lists all requirements found in certification documents. Usage:
	reqtraq list <input_md_filename>
Options:
	<input_md_filename>	Markdown file to be parsed
`

const nextidUsage = `Generates the next requirement id for the given document. Usage:
	reqtraq nextid <input_md_filename>
Options:
	<input_md_filename>	Markdown file to generate the next requirement id for
`

const reportUsage = `
	reportdown 	creates an HTML traceability report from system requirements down to code
	reportissues	creates an HTML report with all issues found in the requirement documents
	reportup 	creates an HTML traceability report from code, to LLRs, to HLRs and to system requirements
Usage:
	reqtraq report<type> [--pfx=<reportfile-prefix>] [--title_filter=<regexp>] [--id_filter=<regexp>]
		[--body_filter=<regexp>] [--since=<start_commid>] [--at=<end_commit>]
		[--certdoc_path=<path>] [--code_path=<path>] [--schema_path=<path_to_attributes_json>]
Options:
	--pfx: path and filename prefix for reports.
	--title_filter: regular expression to filter by requirement title.
	--id_filter: regular expression to filter by requirement id.
	--body_filter: regular expression to filter by requirement body.
	--since: the Git commit SHA-1 representing the start of the range.
	--at: the commit representing the end of the range.
	--certdoc_path: location of certification documents within the current repository
	--code_path: location of source code within the current repository. Default: .
	--schema_path: location of the schema json file. Default: certdocs/attributes.json
`

const validateUsage = `Runs the validation checks for the requirement documents in the current repository. Usage:
	reqtraq validate [--at=<commit>] [--strict] [--certdoc_path=<path>] [--code_path=<path>] [--schema_path=<path_to_attributes_json>]
Options:
	--at: validate this commit rather than the current working copy.
	--strict: if any of the validation checks fail the command will exit with a non-zero code.
	--certdoc_path: location of certification documents within the current repository. Default: certdocs
	--code_path: location of source code within the current repository. Default: .
	--schema_path: location of the schema json file. Default: certdocs/attributes.json
`

const webUsage = `Starts a local web server to facilitate interaction with reqtraq. Usage:
	reqtraq web [--addr=<hostport>]
Options:
	--addr: the ip:port where to serve. Default: localhost:8080.
`

// @llr REQ-TRAQ-SWL-32, REQ-TRAQ-SWL-33, REQ-TRAQ-SWL-34, REQ-TRAQ-SWL-35, REQ-TRAQ-SWL-36
func main() {
	flag.Parse()
	command := flag.Arg(0)
	if command == "" {
		command = "help"
	}

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

	if command == "help" {
		showHelp()
		os.Exit(0)
	}

	// assign global Verbose variable after arguments have been parsed
	linepipes.Verbose = *fVerbose

	switch command {
	case "nextid":
		err := nextId(f)
		if err != nil {
			log.Fatal(err)
		}
	case "list":
		err := list(f)
		if err != nil {
			log.Fatal(err)
		}
	case "reportdown":
		err := reportDown()
		if err != nil {
			log.Fatal(err)
		}
	case "reportup":
		err := reportUp()
		if err != nil {
			log.Fatal(err)
		}
	case "reportissues":
		err := reportIssues()
		if err != nil {
			log.Fatal(err)
		}
	case "web":
		err := Serve(*fAddr)
		if err != nil {
			log.Fatal(err)
		}
	case "validate":
		err := validate()
		if err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Printf(`Invalid command "%s"`, command)
		fmt.Println("")
		fmt.Println(usage)
		os.Exit(1)
	}
}

// buildGraph returns the requirements graph at the specified commit, or the graph for the current files if commit
// is empty. In case the commit is specified, a temporary clone of the repository is created and the path to it is
// returned.
// @llr REQ-TRAQ-SWL-17
func buildGraph(commit string) (*ReqGraph, string, error) {
	if commit == "" {
		rg, err := CreateReqGraph(*fCertdocPath, *fCodePath, *fSchemaPath)
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
	rg, err := CreateReqGraph(*fCertdocPath, *fCodePath, *fSchemaPath)
	if err != nil {
		return rg, dir, errors.Wrap(err, fmt.Sprintf("Failed to create graph in %s", dir))
	}
	if err := os.Chdir(cwd); err != nil {
		return rg, dir, err
	}
	return rg, dir, nil
}

// checkCmdLinePaths validates the command line arguments for alternative paths or files to make sure they exist
// @llr REQ-TRAQ-SWL-35, REQ-TRAQ-SWL-36
func checkCmdLinePaths() error {
	if _, err := os.Stat(filepath.Join(git.RepoPath(), *fCertdocPath)); os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(filepath.Join(git.RepoPath(), *fCodePath)); os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(*fSchemaPath); os.IsNotExist(err) {
		return err
	}
	return nil
}

// createFilterFromCmdLine reads the filter regular expressions from the command line arguments and
// compiles them into a filter structure ready to use
// @llr REQ-TRAQ-SWL-35, REQ-TRAQ-SWL-36
func createFilterFromCmdLine() (ReqFilter, error) {
	filter := ReqFilter{} // Filter for report generation
	var err error
	if len(*fReportTitleFilterString) > 0 {
		filter.TitleRegexp, err = regexp.Compile(*fReportTitleFilterString)
		if err != nil {
			return filter, err
		}
	}
	if len(*fReportIdFilterString) > 0 {
		filter.IDRegexp, err = regexp.Compile(*fReportIdFilterString)
		if err != nil {
			return filter, err
		}
	}
	if len(*fReportBodyFilterString) > 0 {
		filter.BodyRegexp, err = regexp.Compile(*fReportBodyFilterString)
		if err != nil {
			return filter, err
		}
	}
	// TODO can't currently filter on attributes from the command line
	return filter, nil
}

// list parses a single markdown document and lists the requirements to stdout in a short format
// @llr REQ-TRAQ-SWL-33
func list(filename string) error {
	if filename == "" {
		return errors.New("Missing file name")
	}

	reqs, err := ParseCertdoc(filename)
	if err != nil {
		return err
	}
	for _, r := range reqs {
		body := make([]string, 0)
		lines := strings.Split(r.Body, "\n")
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
	return nil
}

// nextId parses a single markdown document for requirements and returns the next available ID
// @llr REQ-TRAQ-SWL-34
func nextId(filename string) error {
	var (
		reqs      []*Req
		nextReqID string
	)

	if filename == "" {
		return errors.New("Missing file name")
	}

	reqs, err := ParseCertdoc(filename)
	if err != nil {
		return err
	}

	if len(reqs) > 0 {
		var (
			lastReq    *Req
			greatestID int = 0
		)
		// infer next req ID from existing req IDs
		for _, r := range reqs {
			parts := ReReqID.FindStringSubmatch(r.ID)
			if parts == nil {
				return fmt.Errorf("Requirement ID invalid: %s", r.ID)
			}
			sequenceNumber := parts[len(parts)-1]
			currentID, err := strconv.Atoi(sequenceNumber)
			if err != nil {
				return fmt.Errorf("Requirement sequence part \"%s\" (%s) not a number:  %s", r.ID, sequenceNumber, err)
			}
			if currentID > greatestID {
				greatestID = currentID
				lastReq = r
			}
		}
		ii := ReReqID.FindStringSubmatchIndex(lastReq.ID)
		nextReqID = fmt.Sprintf("%s%d", lastReq.ID[:ii[len(ii)-2]], greatestID+1)
	} else {
		// infer next (=first) req ID from file name
		fNameWithExt := path.Base(filename)
		extension := filepath.Ext(fNameWithExt)
		fName := fNameWithExt[0 : len(fNameWithExt)-len(extension)]
		fNameComps := strings.Split(fName, "-")
		docType := fNameComps[len(fNameComps)-1]
		reqType, correctFileType := config.DocTypeToReqType[docType]
		if !correctFileType {
			return fmt.Errorf("Document name does not comply with naming convention.")
		}
		nextReqID = "REQ-" + fNameComps[0] + "-" + fNameComps[1] + "-" + reqType + "-001"
	}

	fmt.Println(nextReqID)
	return nil
}

// reportDown creates a requirements graph (and if necessary for comparison a previous graph) and
// generates a top-down html report, showing the implementation for each top-level requirement
// @llr REQ-TRAQ-SWL-35
func reportDown() error {
	err := checkCmdLinePaths()
	if err != nil {
		return err
	}

	var dir string
	rg, dir, err := buildGraph(*fAt)
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	var prg *ReqGraph
	if *fSince != "" {
		var dir string
		prg, dir, err = buildGraph(*fSince)
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
	}

	diffs := rg.ChangedSince(prg)

	of, err := os.Create(*fReportPrefix + "down.html")
	if err != nil {
		return err
	}
	log.Print("Creating ", of.Name(), " (this may take a while)...")
	if err := rg.ReportDown(of); err != nil {
		return err
	}
	of.Close()

	filter, err := createFilterFromCmdLine()
	if err != nil {
		return err
	}
	if !filter.IsEmpty() || diffs != nil {
		of, err := os.Create(*fReportPrefix + "down-filtered.html")
		if err != nil {
			return err
		}
		log.Print("Creating ", of.Name(), " (this may take a while)...")
		if err := rg.ReportDownFiltered(of, &filter, diffs); err != nil {
			return err
		}
		of.Close()
	}

	return nil

}

// reportIssues creates a requirements graph (and if necessary for comparison a previous graph) and
// generates an issues html report, showing any validation problems
// @llr REQ-TRAQ-SWL-36
func reportIssues() error {
	err := checkCmdLinePaths()
	if err != nil {
		return err
	}

	var dir string
	rg, dir, err := buildGraph(*fAt)
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	var prg *ReqGraph
	if *fSince != "" {
		var dir string
		prg, dir, err = buildGraph(*fSince)
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
	}

	diffs := rg.ChangedSince(prg)

	of, err := os.Create(*fReportPrefix + "issues.html")
	if err != nil {
		return err
	}
	log.Print("Creating ", of.Name(), " (this may take a while)...")
	if err := rg.ReportIssues(of); err != nil {
		return err
	}
	of.Close()
	filter, err := createFilterFromCmdLine()
	if err != nil {
		return err
	}
	if !filter.IsEmpty() || diffs != nil {
		of, err := os.Create(*fReportPrefix + "issues-filtered.html")
		if err != nil {
			return err
		}
		log.Print("Creating ", of.Name(), " (this may take a while)...")
		if err := rg.ReportIssuesFiltered(of, &filter, diffs); err != nil {
			return err
		}
		of.Close()
	}

	return nil
}

// reportUp creates a requirements graph (and if necessary for comparison a previous graph) and
// generates a bottom-up html report, showing the top-level requirement for each implemented function
// @llr REQ-TRAQ-SWL-35
func reportUp() error {
	err := checkCmdLinePaths()
	if err != nil {
		return err
	}

	var dir string
	rg, dir, err := buildGraph(*fAt)
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	var prg *ReqGraph
	if *fSince != "" {
		var dir string
		prg, dir, err = buildGraph(*fSince)
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
	}

	diffs := rg.ChangedSince(prg)

	of, err := os.Create(*fReportPrefix + "up.html")
	if err != nil {
		return err
	}
	log.Print("Creating ", of.Name(), " (this may take a while)...")
	if err = rg.ReportUp(of); err != nil {
		return err
	}
	of.Close()

	filter, err := createFilterFromCmdLine()
	if err != nil {
		return err
	}
	if !filter.IsEmpty() || diffs != nil {
		of, err := os.Create(*fReportPrefix + "up-filtered.html")
		if err != nil {
			return err
		}
		log.Print("Creating ", of.Name(), " (this may take a while)...")
		if err := rg.ReportUpFiltered(of, &filter, diffs); err != nil {
			return err
		}
		of.Close()
	}

	return nil
}

// showHelp prints usage information, either for the root command or for a sub-command
// @llr REQ-TRAQ-SWL-32
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
	case "reportup", "reportdown", "reportissues":
		fmt.Println(reportUsage)
	case "validate":
		fmt.Println(validateUsage)
	case "web":
		fmt.Println(webUsage)
	default:
		fmt.Printf("Unknown command '%s'", subCommand)
		fmt.Println(usage)
	}
}

// validate builds the requirement graph, gathering any errors and prints them out. If the strict flag is set return an error.
// @llr REQ-TRAQ-SWL-36
func validate() error {
	err := checkCmdLinePaths()
	if err != nil {
		return err
	}

	rg, dir, err := buildGraph(*fAt)
	os.RemoveAll(dir)
	if err != nil {
		return err
	}

	if len(rg.Errors) > 0 {
		for _, e := range rg.Errors {
			fmt.Println(e)
		}
		if *fStrict {
			return errors.New("ERROR. Validation failed")
		}
		fmt.Println("WARNING. Validation failed")
	} else {
		fmt.Println("Validation passed")
	}
	return nil
}
