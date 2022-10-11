/*
   Functions related to the handling of requirements and code tags.

   The following types and associated methods are implemented:
     ReqGraph - The complete information about a set of requirements and associated code tags.
     Req - A requirement node in the graph of requirements.
     byPosition, byIDNumber and ByFilenameTag - Provides sort functions to order requirements or code,
     ReqFilter - The different parameters used to filter the requirements set.
*/

package main

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
)

type IssueType uint

const (
	IssueTypeInvalidRequirementId IssueType = iota
	IssueTypeInvalidParent
	IssueTypeInvalidRequirementReference
	IssueTypeInvalidRequirementInCode
	IssueTypeMissingRequirementInCode
	IssueTypeMissingAttribute
	IssueTypeUnknownAttribute
	IssueTypeInvalidAttributeValue
	IssueTypeReqTestedButNotImplemented
)

type Issue struct {
	RepoName repos.RepoName
	Path     string
	Line     int
	Error    error
	Type     IssueType
}

// ReqGraph holds the complete information about a set of requirements and associated code tags.
type ReqGraph struct {
	// Reqs contains the requirements by ID.
	Reqs map[string]*Req
	// CodeTags contains the source code functions per file.
	// The keys are paths relative to the git repo path.
	CodeTags map[CodeFile][]*Code
	// Issues which have been found while analyzing the graph.
	// This is extended in multiple places.
	Issues []Issue
	// Holds configuration of reqtraq for all associated repositories
	ReqtraqConfig *config.Config
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
		_, err := repos.GetRepo(repos.BaseRepoName(), repos.RemotePath(repos.BaseRepoPath()), commit, true)
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

// CreateReqGraph returns a graph resulting from parsing the certdocs. The graph includes a list of
// errors found while walking the requirements, code, or resolving the graph.
// The separate returned error indicates if reading the certdocs and code failed.
// @llr REQ-TRAQ-SWL-1
func CreateReqGraph(reqtraqConfig *config.Config) (*ReqGraph, error) {
	rg := &ReqGraph{make(map[string]*Req, 0), make(map[CodeFile][]*Code), make([]Issue, 0), reqtraqConfig}

	// For each repository, we walk through the documents and parse them
	for repoName := range reqtraqConfig.Repos {
		for docIdx := range reqtraqConfig.Repos[repoName].Documents {
			doc := &reqtraqConfig.Repos[repoName].Documents[docIdx]
			if err := rg.addCertdocToGraph(repoName, doc); err != nil {
				return rg, errors.Wrap(err, "Failed parsing certdocs")
			}

			if codeTags, err := ParseCode(repoName, doc); err != nil {
				return rg, errors.Wrap(err, "Failed parsing implementation")
			} else {
				rg.mergeTags(&codeTags)
			}
		}
	}

	// Call resolve to check links between requirements and code
	rg.Issues = append(rg.Issues, rg.resolve()...)

	return rg, nil
}

// Merges all tags from the given map into the ReqGraph instance, potentially replacing them if they
// are already in the requirements graph
// @llr REQ-TRAQ-SWL-8, REQ-TRAQ-SWL-9
func (rg *ReqGraph) mergeTags(tags *map[CodeFile][]*Code) {
	for tagKey := range *tags {
		rg.CodeTags[tagKey] = (*tags)[tagKey]
	}
}

// addCertdocToGraph parses a file for requirements, checks their validity and then adds them along with any errors
// found to the regGraph
// @llr REQ-TRAQ-SWL-27
func (rg *ReqGraph) addCertdocToGraph(repoName repos.RepoName, documentConfig *config.Document) error {
	var reqs []*Req
	var err error
	if reqs, err = ParseMarkdown(repoName, documentConfig); err != nil {
		return fmt.Errorf("Error parsing `%s` in repo `%s`: %v", documentConfig.Path, repoName, err)
	}
	if len(reqs) == 0 {
		return nil
	}

	// sort the requirements so we can check the sequence
	sort.Sort(byIDNumber(reqs))

	isReqPresent := make([]bool, reqs[len(reqs)-1].IDNumber)
	isAsmPresent := make([]bool, reqs[len(reqs)-1].IDNumber)
	nextReqId := 1
	nextAsmId := 1

	for _, r := range reqs {
		var newIssues []Issue
		if r.Variant == ReqVariantRequirement {
			newIssues = r.checkID(documentConfig, nextReqId, isReqPresent)
			nextReqId = r.IDNumber + 1
		} else if r.Variant == ReqVariantAssumption {
			newIssues = r.checkID(documentConfig, nextAsmId, isAsmPresent)
			nextAsmId = r.IDNumber + 1
		}
		if len(newIssues) != 0 {
			rg.Issues = append(rg.Issues, newIssues...)
			continue
		}
		r.RepoName = repoName
		r.Document = documentConfig
		rg.Reqs[r.ID] = r
	}
	return nil
}

// resolve walks the requirements graph and resolves the links between different levels of requirements
// and with code tags. References to requirements within requirements text is checked as well as validity
// of attributes against the schema for their document. Any errors encountered such as links to
// non-existent requirements are returned in a list of issues.
// @llr REQ-TRAQ-SWL-10, REQ-TRAQ-SWL-11, REQ-TRAQ-SWL-67, REQ-TRAQ-SWL-69
func (rg *ReqGraph) resolve() []Issue {
	issues := make([]Issue, 0)

	// Walk the requirements, resolving links and looking for errors
	for _, req := range rg.Reqs {
		if req.IsDeleted() {
			continue
		}

		// Validate requirement Id
		if !req.Document.Schema.Requirements.MatchString(req.ID) {
			issue := Issue{
				Line:     req.Position,
				Path:     req.Document.Path,
				RepoName: req.RepoName,
				Error:    fmt.Errorf("Requirement `%s` in document `%s` does not match required regexp `%s`", req.ID, req.Document.Path, req.Document.Schema.Requirements),
				Type:     IssueTypeInvalidRequirementId,
			}
			issues = append(issues, issue)
		}

		// Validate attributes
		issues = append(issues, req.checkAttributes()...)

		// Validate parent links of requirements
		for _, parentID := range req.ParentIds {
			parent := rg.Reqs[parentID]
			if parent != nil {
				if parent.IsDeleted() {
					issue := Issue{
						Line:     req.Position,
						Path:     req.Document.Path,
						RepoName: req.RepoName,
						Error:    errors.New("Invalid parent of requirement " + req.ID + ": " + parentID + " is deleted."),
						Type:     IssueTypeInvalidParent,
					}
					issues = append(issues, issue)
				}
				parent.Children = append(parent.Children, req)
				req.Parents = append(req.Parents, parent)
			} else {
				issue := Issue{
					Line:     req.Position,
					Path:     req.Document.Path,
					RepoName: req.RepoName,
					Error:    errors.New("Invalid parent of requirement " + req.ID + ": " + parentID + " does not exist."),
					Type:     IssueTypeInvalidParent,
				}
				issues = append(issues, issue)
			}
		}
		// Validate references to requirements in body text
		matches := ReReqID.FindAllStringSubmatchIndex(req.Body, -1)
		for _, ids := range matches {
			reqID := req.Body[ids[0]:ids[1]]
			v, reqFound := rg.Reqs[reqID]
			if !reqFound {
				issue := Issue{
					Line:     req.Position,
					Path:     req.Document.Path,
					RepoName: req.RepoName,
					Error:    fmt.Errorf("Invalid reference to non existent requirement %s in body of %s.", reqID, req.ID),
					Type:     IssueTypeInvalidRequirementReference,
				}
				issues = append(issues, issue)
			} else if v.IsDeleted() {
				issue := Issue{
					Line:     req.Position,
					Path:     req.Document.Path,
					RepoName: req.RepoName,
					Error:    fmt.Errorf("Invalid reference to deleted requirement %s in body of %s.", reqID, req.ID),
					Type:     IssueTypeInvalidRequirementReference,
				}
				issues = append(issues, issue)
			}
		}

		// TODO check for references to missing or deleted requirements in attribute text
	}

	knownSymbols := map[string][]string{}
	llrLoc := map[string]*Code{}

	// Walk the code tags, resolving links and looking for errors
	for _, tags := range rg.CodeTags {
		for _, code := range tags {
			parentIds := knownSymbols[code.Symbol]

			if len(code.ParentIds) == 0 {
				continue
			}

			if code.Symbol == "" {
				continue
			}

			if len(parentIds) == 0 {
				knownSymbols[code.Symbol] = code.ParentIds
				llrLoc[code.Symbol] = code
				continue
			}

			prevLoc := llrLoc[code.Symbol]

			if !reflect.DeepEqual(parentIds, code.ParentIds) {
				issue := Issue{
					Line:     code.Line,
					Path:     code.CodeFile.Path,
					RepoName: code.CodeFile.RepoName,
					Error: fmt.Errorf("LLR declarations differs in %s@%s:%d and %s@%s:%d`.",
						code.Tag, prevLoc.CodeFile.Path, prevLoc.Line, code.Tag, code.CodeFile.Path, code.Line),
					Type: IssueTypeInvalidRequirementInCode,
				}
				issues = append(issues, issue)
			}
		}
	}

	for _, tags := range rg.CodeTags {
		for _, code := range tags {
			parentIds := code.ParentIds
			if code.Symbol != "" {
				parentIds = knownSymbols[code.Symbol]
			}

			if len(parentIds) == 0 && !code.Optional {
				issue := Issue{
					Line:     code.Line,
					Path:     code.CodeFile.Path,
					RepoName: code.CodeFile.RepoName,
					Error:    fmt.Errorf("Function %s@%s:%d has no parents.", code.Tag, code.CodeFile.String(), code.Line),
					Type:     IssueTypeMissingRequirementInCode,
				}
				issues = append(issues, issue)
			}
			for _, parentID := range parentIds {
				if !code.Document.Schema.Requirements.MatchString(parentID) {
					issue := Issue{
						Line:     code.Line,
						Path:     code.CodeFile.Path,
						RepoName: code.CodeFile.RepoName,
						Error: fmt.Errorf("Invalid reference in function %s@%s:%d in repo `%s`, `%s` does not match requirement format in document `%s`.",
							code.Tag, code.CodeFile.Path, code.Line, code.CodeFile.RepoName, parentID, code.Document.Path),
						Type: IssueTypeInvalidRequirementInCode,
					}
					issues = append(issues, issue)
				}

				parent := rg.Reqs[parentID]
				if parent != nil {
					if parent.IsDeleted() {
						issue := Issue{
							Line:     code.Line,
							Path:     code.CodeFile.Path,
							RepoName: code.CodeFile.RepoName,
							Error: fmt.Errorf("Invalid reference in function %s@%s:%d in repo `%s`, %s is deleted.",
								code.Tag, code.CodeFile.Path, code.Line, code.CodeFile.RepoName, parentID),
							Type: IssueTypeInvalidRequirementInCode,
						}
						issues = append(issues, issue)
					}

					parent.Tags = append(parent.Tags, code)
					code.Parents = append(code.Parents, parent)
				} else {
					issue := Issue{
						Line:     code.Line,
						Path:     code.CodeFile.Path,
						RepoName: code.CodeFile.RepoName,
						Error: fmt.Errorf("Invalid reference in function %s@%s:%d in repo `%s`, %s does not exist.",
							code.Tag, code.CodeFile.Path, code.Line, code.CodeFile.RepoName, parentID),
						Type: IssueTypeInvalidRequirementInCode,
					}
					issues = append(issues, issue)
				}
			}
		}
	}

	// Walk through the requirements one last time to ensure that if they are tested they are also implemented.
	// We need to do it at this point, since now the links to the Tags are all set
	for _, req := range rg.Reqs {
		implemented := false
		tested := false
		for _, tag := range req.Tags {
			if tag.CodeFile.Type.Matches(CodeTypeImplementation) {
				implemented = true
			}
			if tag.CodeFile.Type.Matches(CodeTypeTests) {
				tested = true
			}
			if implemented && tested {
				continue
			}
		}

		if !implemented && tested {
			issue := Issue{
				Line:     req.Position,
				Path:     req.Document.Path,
				RepoName: req.RepoName,
				Error:    errors.New("Requirement " + req.ID + " is tested, but it is not implemented."),
				Type:     IssueTypeReqTestedButNotImplemented,
			}
			issues = append(issues, issue)
		}
	}

	if len(issues) > 0 {
		return issues
	}

	for _, req := range rg.Reqs {
		sort.Sort(byPosition(req.Parents))
		sort.Sort(byPosition(req.Children))
	}

	return nil
}

type ReqVariant uint

const (
	ReqVariantRequirement ReqVariant = iota
	ReqVariantAssumption
)

// Req represents a requirement node in the graph of requirements.
type Req struct {
	ID        string // e.g. REQ-TEST-SWL-1
	Variant   ReqVariant
	IDNumber  int // e.g. 1
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
	// Link back to the document where the requirement is defined and the name of the repository
	Document *config.Document
	RepoName repos.RepoName
}

// Changelists generates a list of Phabicator revisions that have affected a requirement
// @llr REQ-TRAQ-SWL-22
func (r *Req) Changelists() map[string]string {
	// TODO(ja): Actually make this return useful information. For now return an empty list
	// To make this work for multiple repositories we need to actually run this against every repo
	m := map[string]string{}
	return m
}

// IsDeleted checks if the requirement title starts with 'DELETED'
// @llr REQ-TRAQ-SWL-23
func (r *Req) IsDeleted() bool {
	return strings.HasPrefix(r.Title, "DELETED")
}

// checkAttributes validates the requirement attributes against the schema from its document,
// returns a list of issues found.
// @llr REQ-TRAQ-SWL-10
func (r *Req) checkAttributes() []Issue {
	var schemaAttributes map[string]*config.Attribute
	switch r.Variant {
	case ReqVariantRequirement:
		schemaAttributes = r.Document.Schema.Attributes
	case ReqVariantAssumption:
		schemaAttributes = r.Document.Schema.AsmAttributes
	}

	var issues []Issue
	var anyAttributes []string
	anyCount := 0

	// Iterate the attribute rules
	for name, attribute := range schemaAttributes {
		if attribute.Type == config.AttributeAny {
			anyAttributes = append(anyAttributes, name)
		}

		reqValue, reqValuePresent := r.Attributes[strings.ToUpper(name)]
		reqValuePresent = reqValuePresent && reqValue != ""

		if !reqValuePresent && attribute.Type == config.AttributeRequired {
			issue := Issue{
				Line:     r.Position,
				Path:     r.Document.Path,
				RepoName: r.RepoName,
				Error:    fmt.Errorf("Requirement '%s' is missing attribute '%s'.", r.ID, name),
				Type:     IssueTypeMissingAttribute,
			}
			issues = append(issues, issue)
		} else if reqValuePresent {
			if attribute.Type == config.AttributeAny {
				anyCount++
			}

			if !attribute.Value.MatchString(reqValue) {
				issue := Issue{
					Line:     r.Position,
					Path:     r.Document.Path,
					RepoName: r.RepoName,
					Error:    fmt.Errorf("Requirement '%s' has invalid value '%s' in attribute '%s'.", r.ID, reqValue, name),
					Type:     IssueTypeInvalidAttributeValue,
				}
				issues = append(issues, issue)
			}
		}
	}

	if len(anyAttributes) > 0 && anyCount == 0 {
		sort.Strings(anyAttributes)
		issue := Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Requirement '%s' is missing at least one of the attributes '%s'.", r.ID, strings.Join(anyAttributes, ",")),
			Type:     IssueTypeMissingAttribute,
		}
		issues = append(issues, issue)
	}

	// Iterate the requirement attributes to check for unknown ones
	for name := range r.Attributes {
		if _, present := schemaAttributes[strings.ToUpper(name)]; !present {
			issue := Issue{
				Line:     r.Position,
				Path:     r.Document.Path,
				RepoName: r.RepoName,
				Error:    fmt.Errorf("Requirement '%s' has unknown attribute '%s'.", r.ID, name),
				Type:     IssueTypeUnknownAttribute,
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// checkID verifies that the requirement is not duplicated
// @llr REQ-TRAQ-SWL-25, REQ-TRAQ-SWL-26, REQ-TRAQ-SWL-28
func (r *Req) checkID(document *config.Document, expectedIDNumber int, isReqPresent []bool) []Issue {
	var issues []Issue
	reqIDComps := strings.Split(r.ID, "-") // results in an array such as [REQ PROJECT REQTYPE 1234]
	// check requirement name, no need to check prefix because it would not have been parsed otherwise
	if reqIDComps[1] != string(document.ReqSpec.Prefix) {
		issue := Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Incorrect project abbreviation for requirement %s. Expected %s, got %s.", r.ID, document.ReqSpec.Prefix, reqIDComps[1]),
			Type:     IssueTypeInvalidRequirementId,
		}
		issues = append(issues, issue)
	}
	if reqIDComps[2] != string(document.ReqSpec.Level) {
		issue := Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Incorrect requirement type for requirement %s. Expected %s, got %s.", r.ID, document.ReqSpec.Level, reqIDComps[2]),
			Type:     IssueTypeInvalidRequirementId,
		}
		issues = append(issues, issue)
	}
	if reqIDComps[3][0] == '0' {
		issue := Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Requirement number cannot begin with a 0: %s. Got %s.", r.ID, reqIDComps[3]),
			Type:     IssueTypeInvalidRequirementId,
		}
		issues = append(issues, issue)
	}

	currentID, err2 := strconv.Atoi(reqIDComps[3])
	if err2 != nil {
		issue := Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Invalid requirement sequence number for %s (failed to parse): %s", r.ID, reqIDComps[3]),
			Type:     IssueTypeInvalidRequirementId,
		}
		issues = append(issues, issue)
	} else {
		if currentID < 1 {
			issue := Issue{
				Line:     r.Position,
				Path:     r.Document.Path,
				RepoName: r.RepoName,
				Error:    fmt.Errorf("Invalid requirement sequence number for %s: first requirement has to start with 001.", r.ID),
				Type:     IssueTypeInvalidRequirementId,
			}
			issues = append(issues, issue)
		} else {
			if isReqPresent[currentID-1] {
				issue := Issue{
					Line:     r.Position,
					Path:     r.Document.Path,
					RepoName: r.RepoName,
					Error:    fmt.Errorf("Invalid requirement sequence number for %s, is duplicate.", r.ID),
					Type:     IssueTypeInvalidRequirementId,
				}
				issues = append(issues, issue)
			} else {
				if currentID != expectedIDNumber {
					issue := Issue{
						Line:     r.Position,
						Path:     r.Document.Path,
						RepoName: r.RepoName,
						Error:    fmt.Errorf("Invalid requirement sequence number for %s: missing requirements in between. Expected ID Number %d.", r.ID, expectedIDNumber),
						Type:     IssueTypeInvalidRequirementId,
					}
					issues = append(issues, issue)
				}
			}
			isReqPresent[currentID-1] = true
		}
	}

	return issues
}

type AttributeRule struct {
	Filter   *regexp.Regexp // holds a regex which matches which requirement IDs this rule applies to
	Required bool           // indicates if this attribute is mandatory
	Any      bool           // indicates if this attribute, or another with this flag set, is mandatory
	Value    *regexp.Regexp // regex which matches valid values for the attribute
}

// byPosition provides sort functions to order requirements by their Position value
type byPosition []*Req

// @llr REQ-TRAQ-SWL-45
func (a byPosition) Len() int { return len(a) }

// @llr REQ-TRAQ-SWL-45
func (a byPosition) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// @llr REQ-TRAQ-SWL-45
func (a byPosition) Less(i, j int) bool { return a[i].Position < a[j].Position }

// byIDNumber provides sort functions to order requirements by their IDNumber value
type byIDNumber []*Req

// @llr REQ-TRAQ-SWL-46
func (a byIDNumber) Len() int { return len(a) }

// @llr REQ-TRAQ-SWL-46
func (a byIDNumber) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// @llr REQ-TRAQ-SWL-46
func (a byIDNumber) Less(i, j int) bool { return a[i].IDNumber < a[j].IDNumber }

// byFilenameTag provides sort functions to order code by their repo name, then path value, and then line number
type byFilenameTag []*Code

// @llr REQ-TRAQ-SWL-47
func (a byFilenameTag) Len() int { return len(a) }

// @llr REQ-TRAQ-SWL-47
func (a byFilenameTag) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// @llr REQ-TRAQ-SWL-47
func (a byFilenameTag) Less(i, j int) bool {
	switch strings.Compare(string(a[i].CodeFile.RepoName), string(a[j].CodeFile.RepoName)) {
	case -1:
		return true
	case 0:
		switch strings.Compare(a[i].CodeFile.Path, a[j].CodeFile.Path) {
		case -1:
			return true
		case 0:
			return a[i].Line < a[j].Line
		}
		return false
	}
	return false
}

// ReqFilter holds the different parameters used to filter the requirements set.
type ReqFilter struct {
	IDRegexp           *regexp.Regexp
	TitleRegexp        *regexp.Regexp
	BodyRegexp         *regexp.Regexp
	AnyAttributeRegexp *regexp.Regexp
	AttributeRegexp    map[string]*regexp.Regexp
}

// IsEmpty returns whether the filter has no restriction.
// @llr REQ-TRAQ-SWL-20, REQ-TRAQ-SWL-21
func (f ReqFilter) IsEmpty() bool {
	return f.IDRegexp == nil && f.TitleRegexp == nil &&
		f.BodyRegexp == nil && f.AnyAttributeRegexp == nil &&
		len(f.AttributeRegexp) == 0
}
