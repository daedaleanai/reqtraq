package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/git"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/pkg/errors"
)

type RequirementStatus int

const (
	NOT_STARTED RequirementStatus = iota // req does not have any children, unless code level
	STARTED                              // req does have children but incomplete
	COMPLETED                            // graph complete
)

var reqStatusToString = map[RequirementStatus]string{
	NOT_STARTED: "NOT STARTED",
	STARTED:     "STARTED",
	COMPLETED:   "COMPLETED",
}

func (rs RequirementStatus) String() string { return reqStatusToString[rs] }

var (
	reDiffRev = regexp.MustCompile(`Differential Revision:\s(.*)\s`)
)

// Req represenents a Requirement Node in the graph of Requirements.
// The Attributes map has potential elements:
//  rationale safety_impact verification urgent important mode provenance
type Req struct {
	ID       string
	IDNumber int
	Level    config.RequirementLevel
	// Path identifies the file this was found in relative to repo root.
	Path      string
	ParentIds []string
	// Parents holds the parent requirements.
	Parents []*Req
	// Children holds the children requirements.
	Children []*Req
	// Tags holds the associated code functions.
	Tags  []*Code
	Title string
	Body  string
	// Attributes of the requirement by uppercase name.
	Attributes map[string]string
	Position   int
	Status     RequirementStatus
}

// Returns the requirement type for the given requirement, which is one of SYS, SWH, SWL, HWH, HWL or the empty string if
// the request is not initialized.
func (r *Req) ReqType() string {
	parts := ReReqID.FindStringSubmatch(r.ID)
	if len(parts) == 0 {
		return ""
	}
	return parts[2]
}

// IsDeleted checks if the requirement title starts with 'DELETED'
// @llr REQ-TRAQ-SWL-17
func (r *Req) IsDeleted() bool {
	return strings.HasPrefix(r.Title, "DELETED")
}

func (r *Req) CheckAttributes(as []map[string]string) []error {
	var errs []error
	if r.IsDeleted() {
		return errs
	}
	// Iterate the attribute specs.
	for _, a := range as {
		for k, v := range a {
			switch k {
			case "name":
				if _, ok := r.Attributes[strings.ToUpper(v)]; !ok {
					if !(r.Level == config.SYSTEM && strings.ToUpper(v) == "PARENTS") {
						errs = append(errs, fmt.Errorf("Requirement '%s' is missing attribute '%s'.", r.ID, v))
					}
				}
			case "value":
				aName := strings.ToUpper(a["name"])
				if _, ok := r.Attributes[aName]; ok {
					// attribute exists so needs to be valid
					expr, err := regexp.Compile(v) // TODO(dh) move out so only computed once for each value
					if err != nil {
						log.Fatal(err)
					}
					if !expr.MatchString(r.Attributes[aName]) {
						errs = append(errs, fmt.Errorf("Requirement '%s' has invalid value '%s' in attribute '%s'. Expected %s.", r.ID, r.Attributes[aName], aName, v))
					}
				}
			}
		}
	}
	return errs
}

// @llr REQ-TRAQ-SWL-9
func (r *Req) Changelists() map[string]string {
	m := map[string]string{}
	if r.Level == config.LOW {
		var paths []string
		for _, c := range r.Children {
			paths = append(paths, c.Path)
		}
		urls := changelistUrlsForFilepaths(paths)
		for _, url := range urls {
			fields := strings.Split(url, "/")
			m[fields[len(fields)-1]] = url
		}
	}
	return m
}

func changelistUrlsForFilepaths(filepaths []string) []string {
	var urls []string
	for _, path := range filepaths {
		urls = append(urls, changelistUrlsForFilepath(path)...)
	}
	return urls
}

func changelistUrlsForFilepath(filepath string) []string {
	res, err := linepipes.All(linepipes.Run("git", "-C", path.Dir(filepath), "log", filepath))
	if err != nil {
		log.Fatal(err)
	}

	matches := reDiffRev.FindAllStringSubmatch(res, -1)
	if len(matches) < 1 {
		log.Printf("Could not extract differential revision for file: %s", filepath)
		log.Println("Newly added?")
	}

	var urls []string
	for _, m := range matches {
		if len(m) != 2 {
			log.Fatal("Count not extract changelist substring for filepath: ", filepath)
		}
		urls = append(urls, m[1])
	}

	return urls
}

// Code represents a Code Node in the graph of Requirements.
type Code struct {
	// Path is the code file this was found in relative to repo root.
	Path string
	// Tag is the name of the function.
	Tag string
	// Line number where the function starts.
	Line int
	// The comment above the function.
	Comment []string
	// FileHash is the sha1 of the contents.
	FileHash  string
	ParentIds []string
	Parents   []*Req
}

var reLLRReference = regexp.MustCompile(fmt.Sprintf(`\s@llr (%s)\z`, reReqIdStr))

func (code *Code) URL() string {
	return fmt.Sprintf("/code/%s#L%d", code.Path, code.CommentLine())
}

func (code *Code) CommentLine() int {
	return code.Line - len(code.Comment)
}

func (code *Code) ReqIDsInComment() []string {
	refs := make([]string, 0)
	for _, s := range code.Comment {
		if parts := reLLRReference.FindStringSubmatch(s); len(parts) > 0 {
			refs = append(refs, parts[1])
		}
	}
	return refs
}

// A ReqGraph maps IDs and code files paths to Req structures.
// @llr REQ-TRAQ-SWL-15
type reqGraph struct {
	// Reqs contains the requirements by ID.
	Reqs map[string]*Req
	// CodeTags contains the source code functions per file.
	// The keys are paths relative to the git repo path.
	CodeTags map[string][]*Code
	// CodeFiles are the paths to the discovered code files,
	// relative to the git repo path.
	CodeFiles []string
	// Errors which have been found while analyzing the graph.
	// This is extended in multiple places.
	Errors []error
}

// CreateReqGraph returns a graph resulting from parsing the certdocs,
// containing errors found while walking the requirements, code, or resolving
// the graph.
// The separate returned error specifies how reading the certdocs and code
// failed, if it did.
func CreateReqGraph(certdocsPath, codePath string) (*reqGraph, error) {
	rg := &reqGraph{make(map[string]*Req, 0), nil, make([]string, 0), make([]error, 0)}

	filesListFile, err := ioutil.TempFile("", "list-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(filesListFile.Name())

	// Walk through the documents.
	err = filepath.Walk(filepath.Join(git.RepoPath(), certdocsPath),
		func(fileName string, info os.FileInfo, err error) error {
			switch strings.ToLower(path.Ext(fileName)) {
			case ".md":
				parseErrs, err := parseCertdocToGraph(fileName, rg)
				if err != nil {
					return err
				}
				rg.Errors = append(rg.Errors, parseErrs...)
			case ".go":
				if _, err := filesListFile.WriteString(fileName); err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return rg, errors.Wrap(err, "failed walking certdocs")
	}

	// Find the code files.
	rg.CodeFiles, err = findCodeFiles(git.RepoPath(), codePath)
	if err != nil {
		return rg, errors.Wrap(err, "failed to find code files")
	}

	// Discover the code procedures.
	rg.CodeTags, err = tagCode(rg.CodeFiles)
	if err != nil {
		return rg, errors.Wrap(err, "failed to tag code")
	}

	// Annotate the code procedures with the associated comment.
	if err := rg.parseComments(); err != nil {
		return rg, errors.Wrap(err, "failed walking code")
	}

	rg.Errors = append(rg.Errors, rg.Resolve()...)

	return rg, nil
}

// relativePathToRepo returns filePath relative to repoPath by
// removing the path to the repository from filePath
func relativePathToRepo(filePath, repoPath string) (string, error) {
	if filePath[:1] != "/" {
		// Already a relative path.
		return filePath, nil
	}
	fields := strings.SplitN(filePath, repoPath, 2)
	if len(fields) < 2 {
		return "", fmt.Errorf("malformed code file path: %s not in %s", filePath, repoPath)
	}
	res := fields[1]
	if res[:1] == "/" {
		res = res[1:]
	}
	return res, nil
}

func (rg *reqGraph) AddReq(req *Req, path string) error {
	if v := rg.Reqs[req.ID]; v != nil {
		return fmt.Errorf("Requirement %s in %s already defined in %s", req.ID, path, v.Path)
	}
	req.Path = strings.TrimPrefix(path, git.RepoPath())

	rg.Reqs[req.ID] = req
	return nil
}

func (rg *reqGraph) CheckAttributes(reportConf JsonConf, filter *ReqFilter, diffs map[string][]string) ([]error, error) {
	var errs []error
	for _, req := range rg.Reqs {
		if req.Matches(filter, diffs) {
			errs = append(errs, req.CheckAttributes(reportConf.Attributes)...)
		}
	}
	return errs, nil
}

// @llr REQ-TRAQ-SWL-4
func (rg *reqGraph) checkReqReferences(certdocPath string) ([]error, error) {
	reParents := regexp.MustCompile(`Parents: REQ-`)

	errs := make([]error, 0)

	err := filepath.Walk(filepath.Join(git.RepoPath(), certdocPath),
		func(fileName string, info os.FileInfo, err error) error {
			r, err := os.Open(fileName)
			if err != nil {
				return err
			}

			scan := bufio.NewScanner(r)
			for lno := 1; scan.Scan(); lno++ {
				line := scan.Text()
				// parents have alreay been checked in Resolve(), and we don't throw an eror at the place where the deleted req is defined
				discardRefToDeleted := reParents.MatchString(line) || ReReqDeleted.MatchString(line)
				parmatch := ReReqID.FindAllStringSubmatchIndex(line, -1)
				for _, ids := range parmatch {
					reqID := line[ids[0]:ids[1]]
					v, reqFound := rg.Reqs[reqID]
					if !reqFound {
						errs = append(errs, fmt.Errorf("Invalid reference to inexistent requirement %s in %s:%d", reqID, fileName, lno))
					} else if v.IsDeleted() && !discardRefToDeleted {
						errs = append(errs, fmt.Errorf("Invalid reference to deleted requirement %s in %s:%d", reqID, fileName, lno))
					}
				}
			}
			return nil
		})

	if err != nil {
		return nil, err
	}

	return errs, nil
}

// @llr REQ-TRAQ-SWL-17
func (rg *reqGraph) Resolve() []error {
	errs := make([]error, 0)

	for _, req := range rg.Reqs {
		if len(req.ParentIds) == 0 && !(req.Level == config.SYSTEM || req.IsDeleted()) {
			errs = append(errs, errors.New("Requirement "+req.ID+" in file "+req.Path+" has no parents."))
		}
		for _, parentID := range req.ParentIds {
			parent := rg.Reqs[parentID]
			if parent != nil {
				if parent.IsDeleted() && !req.IsDeleted() {
					errs = append(errs, errors.New("Invalid parent of requirement "+req.ID+": "+parentID+" is deleted."))
				}
				parent.Children = append(parent.Children, req)
				req.Parents = append(req.Parents, parent)
			} else {
				errs = append(errs, errors.New("Invalid parent of requirement "+req.ID+": "+parentID+" does not exist."))
			}
		}
	}

	errs = rg.resolveCodeTags(errs)
	if len(errs) > 0 {
		return errs
	}

	for _, req := range rg.Reqs {
		sort.Sort(byPosition(req.Parents))
		sort.Sort(byPosition(req.Children))
	}

	return nil
}

func (rg *reqGraph) resolveCodeTags(errs []error) []error {
	for _, tags := range rg.CodeTags {
		for i := range tags {
			code := tags[i]
			code.ParentIds = code.ReqIDsInComment()
			if len(code.ParentIds) == 0 {
				errs = append(errs, errors.New("Function "+code.Tag+" in file "+code.Path+" has no parents."))
			}
			for _, parentID := range code.ParentIds {
				parent := rg.Reqs[parentID]
				if parent != nil {
					if parent.IsDeleted() {
						errs = append(errs, errors.New("Invalid reference in file "+code.Path+" function "+code.Tag+": "+parentID+" is deleted."))
					}
					if parent.Level == config.LOW {
						parent.Tags = append(parent.Tags, code)
						code.Parents = append(code.Parents, parent)
					} else {
						errs = append(errs, errors.New("Invalid reference in file "+code.Path+" function "+code.Tag+": "+parentID+" must be a Software Low-Level Requirement."))
					}
				} else {
					errs = append(errs, errors.New("Invalid reference in file "+code.Path+" function "+code.Tag+": "+parentID+" does not exist."))
				}
			}
		}
	}
	return errs
}

// OrdsByPosition returns the SYSTEM requirements which don't have any parent,
// ordered by position.
func (rg reqGraph) OrdsByPosition() []*Req {
	var r []*Req
	for _, v := range rg.Reqs {
		if v.Level == config.SYSTEM && len(v.ParentIds) == 0 {
			r = append(r, v)
		}
	}
	sort.Sort(byPosition(r))
	return r
}

func (rg *reqGraph) ReqsWithInvalidRequirementsByPosition() []*Req {
	var r []*Req

	return r
}

type byPosition []*Req

func (a byPosition) Len() int           { return len(a) }
func (a byPosition) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPosition) Less(i, j int) bool { return a[i].Position < a[j].Position }

type byIDNumber []*Req

func (a byIDNumber) Len() int           { return len(a) }
func (a byIDNumber) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byIDNumber) Less(i, j int) bool { return a[i].IDNumber < a[j].IDNumber }

type byFilenameTag []*Code

func (a byFilenameTag) Len() int      { return len(a) }
func (a byFilenameTag) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byFilenameTag) Less(i, j int) bool {
	switch strings.Compare(a[i].Path, a[j].Path) {
	case -1:
		return true
	case 0:
		return a[i].Tag < a[j].Tag
	}
	return false
}

func parseCertdocToGraph(fileName string, graph *reqGraph) ([]error, error) {
	reqs, err := ParseCertdoc(fileName)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", fileName, err)
	}

	// sort the requirements so we can check the sequence
	sort.Sort(byIDNumber(reqs))

	isReqPresent := make([]bool, reqs[len(reqs)-1].IDNumber)
	NextId := 1

	var errs []error
	for i, r := range reqs {
		errs2 := lintReq(fileName, NextId, isReqPresent, r)
		NextId = r.IDNumber + 1
		if len(errs2) != 0 {
			errs = append(errs, errs2...)
			continue
		}
		r.Position = i
		graph.AddReq(r, fileName)
	}
	return errs, nil
}

// lintReq is called for each requirement while building the req graph
// @llr REQ-TRAQ-SWL-3
// @llr REQ-TRAQ-SWL-5
func lintReq(fileName string, expectedIDNumber int, isReqPresent []bool, r *Req) []error {
	// extract file name without extension
	fNameWithExt := path.Base(fileName)
	extension := filepath.Ext(fNameWithExt)
	fName := fNameWithExt[0 : len(fNameWithExt)-len(extension)]

	// figure out req type from doc type
	fNameComps := strings.Split(fName, "-")
	docType := fNameComps[len(fNameComps)-1]
	reqType := config.DocTypeToReqType[docType]

	var errs []error
	reqIDComps := strings.Split(r.ID, "-") // results in an array such as [REQ PROJECT REQTYPE 1234]
	// check requirement name
	if reqIDComps[0] != "REQ" {
		errs = append(errs, fmt.Errorf("Incorrect requirement name %s. Every requirement needs to start with REQ, got %s.", r.ID, reqIDComps[0]))
	}
	if reqIDComps[1] != fNameComps[0] {
		errs = append(errs, fmt.Errorf("Incorrect project abbreviation for requirement %s. Expected %s, got %s.", r.ID, fNameComps[0], reqIDComps[1]))
	}
	if reqIDComps[2] != reqType {
		errs = append(errs, fmt.Errorf("Incorrect requirement type for requirement %s. Expected %s, got %s.", r.ID, reqType, reqIDComps[2]))
	}
	if reqIDComps[3][0] == '0' {
		errs = append(errs, fmt.Errorf("Requirement number cannot begin with a 0: %s. Got %s.", r.ID, reqIDComps[3]))
	}

	currentID, err2 := strconv.Atoi(reqIDComps[3])
	if err2 != nil {
		errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s (failed to parse): %s", r.ID, reqIDComps[3]))
	} else {
		if currentID < 1 {
			errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s: first requirement has to start with 001.", r.ID))
		} else {
			if isReqPresent[currentID-1] {
				errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s, is duplicate.", r.ID))
			} else {
				if currentID != expectedIDNumber {
					errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s: missing requirements in between. Expected ID Number %d.", r.ID, expectedIDNumber))
				}
			}
			isReqPresent[currentID-1] = true
		}
	}

	return errs
}

type ReqFilter struct {
	IDRegexp           *regexp.Regexp
	TitleRegexp        *regexp.Regexp
	BodyRegexp         *regexp.Regexp
	AnyAttributeRegexp *regexp.Regexp
	AttributeRegexp    map[string]*regexp.Regexp
}

// IsEmpty returns whether the filter has no restriction.
func (f ReqFilter) IsEmpty() bool {
	return f.IDRegexp == nil && f.TitleRegexp == nil &&
		f.BodyRegexp == nil && f.AnyAttributeRegexp == nil &&
		len(f.AttributeRegexp) == 0
}

// Matches returns true if the requirement matches the filter AND its ID is
// in the diffs map, if any.
// @llr REQ-TRAQ-SWL-12
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

func NextId(f string) (string, error) {
	var (
		reqs      []*Req
		nextReqID string
	)

	reqs, err := ParseCertdoc(f)
	if err != nil {
		return "", err
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
				return "", fmt.Errorf("Requirement ID invalid: %s", r.ID)
			}
			sequenceNumber := parts[len(parts)-1]
			currentID, err := strconv.Atoi(sequenceNumber)
			if err != nil {
				return "", fmt.Errorf("Requirement sequence part \"%s\" (%s) not a number:  %s", r.ID, sequenceNumber, err)
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
		fNameWithExt := path.Base(f)
		extension := filepath.Ext(fNameWithExt)
		fName := fNameWithExt[0 : len(fNameWithExt)-len(extension)]
		fNameComps := strings.Split(fName, "-")
		docType := fNameComps[len(fNameComps)-1]
		reqType, correctFileType := config.DocTypeToReqType[docType]
		if !correctFileType {
			return "", fmt.Errorf("Document name does not comply with naming convention.")
		}
		nextReqID = "REQ-" + fNameComps[0] + "-" + fNameComps[1] + "-" + reqType + "-001"
	}

	return nextReqID, nil
}
