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
	"regexp"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
)

var (
	fReportPrefix            = flag.String("pfx", "./req-", "Path and filename prefix for reports.")
	fReportTitleFilterString = flag.String("title_filter", "", "Regular expression to filter by requirement title.")
	fReportIdFilterString    = flag.String("id_filter", "", "Regular expression to filter by requirement id.")
	fReportBodyFilterString  = flag.String("body_filter", "", "Regular expression to filter by requirement body.")
	fAddr                    = flag.String("addr", ":8080", "The ip:port where to serve.")
	fSince                   = flag.String("since", "", "The commit representing the start of the range.")
	fAt                      = flag.String("at", "", "The commit representing the end of the range.")
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
Argument:
	<input_md_filename>	Markdown file to be parsed`

const nextidUsage = `Generates the next requirement id for the given document. Usage:
	reqtraq nextid <input_md_filename>
Argument:
	<input_md_filename>	Markdown file to generate the next requirement id for`

const reportUsage = `
	reportdown 	creates an HTML traceability report from system requirements down to code
	reportissues	creates an HTML report with all issues found in the requirement documents
	reportup 	creates an HTML traceability report from code, to LLRs, to HLRs and to system requirements
Usage:
	reqtraq report<type> [--pfx=<reportfile-prefix>] [--title_filter=<regexp>] [--id_filter=<regexp>]
		[--body_filter=<regexp>] [--since=<start_commid>] [--at=<end_commit>]
Options:
	--pfx: path and filename prefix for reports.
	--title_filter: regular expression to filter by requirement title.
	--id_filter: regular expression to filter by requirement id.
	--body_filter: regular expression to filter by requirement body.
	--since: the Git commit SHA-1 representing the start of the range.
	--at: the commit representing the end of the range.`

const validateUsage = `Runs the validation checks for the requirement documents in the current repository. Usage:
	reqtraq validate [--at=<commit>] [--strict]
Options:
	--at: validate this commit rather than the current working copy.
	--strict: if any of the validation checks fail the command will exit with a non-zero code.`

const webUsage = `Starts a local web server to facilitate interaction with reqtraq. Usage:
	reqtraq web [--addr=<hostport>]
Options:
	--addr: the ip:port where to serve. Default: localhost:8080.`

var reCertdoc = regexp.MustCompile(`^(\w+)-(\d+)-(\w+)$`)

// @llr REQ-TRAQ-SWL-32, REQ-TRAQ-SWL-33, REQ-TRAQ-SWL-34, REQ-TRAQ-SWL-35, REQ-TRAQ-SWL-36
func main() {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	defer repos.CleanupTemporaryDirectories()

	flag.Parse()

	command := flag.Arg(0)
	if command == "" {
		command = "help"
	}

	// check to see if the command has a second parameter, e.g. list <filename>
	filename := ""
	remainingArgs := flag.Args()
	if len(remainingArgs) > 1 {
		if !strings.HasPrefix(remainingArgs[1], "-") {
			filename = remainingArgs[1]
		}
		// See maybe there are more flags after the `action`.
		os.Args = append(os.Args[:1], remainingArgs[1:]...)
		flag.Parse()
	}

	// assign global Verbose variable after arguments have been parsed
	linepipes.Verbose = *fVerbose

	// Register BaseRepository so that it is always accessible afterwards
	baseRepoPath := repos.BaseRepoPath()
	repos.RegisterRepository(baseRepoPath)

	reqtraqConfig, err := config.ParseConfig(baseRepoPath)
	if err != nil {
		log.Fatal("Error parsing `reqtraq_config.json` file in current repo:", err)
	}

	switch command {
	case "nextid":
		err = nextId(filename, &reqtraqConfig)
	case "list":
		err = list(filename, &reqtraqConfig)
	case "reportdown":
		err = reportDown(&reqtraqConfig)
	case "reportup":
		err = reportUp(&reqtraqConfig)
	case "reportissues":
		err = reportIssues(&reqtraqConfig)
	case "web":
		err = Serve(*fAddr, &reqtraqConfig)
	case "validate":
		err = validate(&reqtraqConfig)
	case "help":
		showHelp()
		os.Exit(0)
	default:
		fmt.Printf(`Invalid command "%s"`, command)
		fmt.Println("")
		fmt.Println(usage)
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
}

// buildGraph returns the requirements graph at the specified commit, or the graph for the current files if commit
// is empty. In case the commit is specified, a temporary clone of the repository is created and the path to it is
// returned.
// @llr REQ-TRAQ-SWL-17
func buildGraph(commit string, reqtraqConfig *config.Config) (*ReqGraph, error) {
	if commit != "" {
		// Override the current repository to get a different revision. This will create a clone
		// of the repo with the specified revision and it will be always used after this call for
		// the base repo
		_, _, err := repos.GetRepo(repos.RemotePath(repos.BaseRepoPath()), commit, true)
		if err != nil {
			return nil, err
		}

		// Also override reqtraq configuration... as they are different repos
		overridenConfig, err := config.ParseConfig(repos.BaseRepoPath())
		if err != nil {
			return nil, err
		}

		// Create the req graph with the new repository
		rg, err := CreateReqGraph(&overridenConfig)
		if err != nil {
			return rg, errors.Wrap(err, fmt.Sprintf("Failed to create graph"))
		}
		return rg, nil
	}

	// Create the req graph with the new repository
	rg, err := CreateReqGraph(reqtraqConfig)
	if err != nil {
		return rg, errors.Wrap(err, fmt.Sprintf("Failed to create graph"))
	}
	return rg, nil
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
func list(filename string, reqtraqConfig *config.Config) error {
	if filename == "" {
		return errors.New("Missing file name")
	}

	var repoName repos.RepoName
	var certdocConfig *config.Document
	if repoName, certdocConfig = reqtraqConfig.FindCertdoc(filename); certdocConfig == nil {
		return fmt.Errorf("Could not find document `%s` in the list of documents", filename)
	}
	reqs, err := ParseMarkdown(repoName, certdocConfig)
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
func nextId(filename string, reqtraqConfig *config.Config) error {
	var (
		reqs          []*Req
		greatestReqID int = 0
		greatestAsmID int = 0
	)

	if filename == "" {
		return errors.New("No filename was given. Call reqtraq with a certdoc path from the root of the repository")
	}

	var repoName repos.RepoName
	var certdocConfig *config.Document
	if repoName, certdocConfig = reqtraqConfig.FindCertdoc(filename); certdocConfig == nil {
		return fmt.Errorf("Could not find document `%s` in the list of documents", filename)
	}

	reqs, err := ParseMarkdown(repoName, certdocConfig)
	if err != nil {
		return err
	}

	// ParseCertdoc validated the filename format, so no need to validate again
	filenameParts := reCertdoc.FindStringSubmatch(strings.TrimSuffix(path.Base(filename), ".md"))

	// count existing REQ and ASM IDs
	for _, r := range reqs {
		if r.Prefix == "REQ" && r.IDNumber > greatestReqID {
			greatestReqID = r.IDNumber
		} else if r.Prefix == "ASM" && r.IDNumber > greatestAsmID {
			greatestAsmID = r.IDNumber
		}
	}

	fmt.Printf("REQ-%s-%s-%d\n", filenameParts[1], config.DocTypeToReqType[filenameParts[3]], greatestReqID+1)

	// don't bother reporting assumptions if none are defined yet
	if greatestAsmID > 0 {
		fmt.Printf("ASM-%s-%s-%d\n", filenameParts[1], config.DocTypeToReqType[filenameParts[3]], greatestAsmID+1)
	}

	return nil
}

// reportDown creates a requirements graph (and if necessary for comparison a previous graph) and
// generates a top-down html report, showing the implementation for each top-level requirement
// @llr REQ-TRAQ-SWL-35
func reportDown(reqtraqConfig *config.Config) error {
	rg, err := buildGraph(*fAt, reqtraqConfig)
	if err != nil {
		return err
	}

	var prg *ReqGraph
	if *fSince != "" {
		prg, err = buildGraph(*fSince, reqtraqConfig)
		if err != nil {
			return err
		}
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
func reportIssues(reqtraqConfig *config.Config) error {
	rg, err := buildGraph(*fAt, reqtraqConfig)
	if err != nil {
		return err
	}

	var prg *ReqGraph
	if *fSince != "" {
		prg, err = buildGraph(*fSince, reqtraqConfig)
		if err != nil {
			return err
		}
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
func reportUp(reqtraqConfig *config.Config) error {
	rg, err := buildGraph(*fAt, reqtraqConfig)
	if err != nil {
		return err
	}

	var prg *ReqGraph
	if *fSince != "" {
		prg, err = buildGraph(*fSince, reqtraqConfig)
		if err != nil {
			return err
		}
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
func validate(reqtraqConfig *config.Config) error {
	rg, err := buildGraph(*fAt, reqtraqConfig)
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
