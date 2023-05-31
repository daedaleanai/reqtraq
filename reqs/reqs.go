/*
   Functions related to the handling of requirements and code tags.

   The following types and associated methods are implemented:
     ReqGraph - The complete information about a set of requirements and associated code tags.
     Req - A requirement node in the graph of requirements.
     byPosition, byIDNumber and ByFilenameTag - Provides sort functions to order requirements or code,
     ReqFilter - The different parameters used to filter the requirements set.
*/

package reqs

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/code"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/diagnostics"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
)

// OrdsByPosition returns the SYSTEM requirements which don't have any parent, ordered by position.
// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-20
func (rg ReqGraph) OrdsByPosition() []*Req {
	var r []*Req
	for _, v := range rg.Reqs {
		if v.Document.LinkSpecs == nil && len(v.ParentIds) == 0 {
			r = append(r, v)
		}
	}
	sort.Sort(byPosition(r))
	return r
}

// BuildGraph returns a graph resulting from parsing the certdocs. The graph includes a list of
// errors found while walking the requirements, code, or resolving the graph.
// The separate returned error indicates if reading the certdocs and code failed.
// @llr REQ-TRAQ-SWL-1
func BuildGraph(reqtraqConfig *config.Config) (*ReqGraph, error) {
	rg := &ReqGraph{make(map[string]*Req, 0), make(map[code.CodeFile][]*code.Code), make([]diagnostics.Issue, 0), reqtraqConfig}

	// For each repository, we walk through the documents and parse them
	for repoName := range reqtraqConfig.Repos {
		for docIdx := range reqtraqConfig.Repos[repoName].Documents {
			doc := &reqtraqConfig.Repos[repoName].Documents[docIdx]
			if err := rg.addCertdocToGraph(repoName, doc); err != nil {
				return rg, errors.Wrap(err, "Failed parsing certdocs")
			}

			if codeTags, err := code.ParseCode(repoName, doc); err != nil {
				return rg, errors.Wrap(err, "Failed parsing implementation")
			} else {
				rg.mergeTags(&codeTags)
			}
		}
	}

	// Call Resolve to check links between requirements and code
	rg.Issues = append(rg.Issues, rg.Resolve()...)

	return rg, nil
}

// Merges all tags from the given map into the ReqGraph instance, potentially replacing them if they
// are already in the requirements graph
// @llr REQ-TRAQ-SWL-8, REQ-TRAQ-SWL-9
func (rg *ReqGraph) mergeTags(tags *map[code.CodeFile][]*code.Code) {
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
		var newIssues []diagnostics.Issue
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

// Code may be declared many times and defined at least once per binary. To avoid having to repeat
// the same llr in all declarations and definitions this function deduplicates entries and keeps
// makes sure that all code tags use the same llr. If more than one tag with the same symbol uses a
// different LLR this triggers an issue that is reported.
// @llr REQ-TRAQ-SWL-10, REQ-TRAQ-SWL-11, REQ-TRAQ-SWL-67, REQ-TRAQ-SWL-69
func (rg *ReqGraph) deduplicateCodeSymbols() ([]diagnostics.Issue, func(doc, symbol string) []string) {
	issues := make([]diagnostics.Issue, 0)

	// Deduplication must only happen per requirement document. One document may provide a declaration,
	// which links a requirement in that document, while a different item may provide a definition and
	// use a different requirements. Sometimes this means they may be different definitions for the
	// same symbols even within the same project linking to different requirements.

	// Map of parentIds each document and symbol. First key is the document, second key is the symbol
	linksMap := map[string]map[string][]string{}
	llrLoc := map[string]map[string]*code.Code{}

	getParentIdsForSymbolInDocument := func(doc, symbol string) []string {
		if _, ok := linksMap[doc]; !ok {
			linksMap[doc] = make(map[string][]string)
		}
		return linksMap[doc][symbol]
	}

	setParentIdsForSymbolInDocument := func(doc, symbol string, links []code.ReqLink) {
		if _, ok := linksMap[doc]; !ok {
			linksMap[doc] = make(map[string][]string)
		}
		ids := []string{}
		for _, link := range links {
			ids = append(ids, link.Id)
		}
		linksMap[doc][symbol] = ids
	}

	getLlrLocForSymbolInDocument := func(doc, symbol string) *code.Code {
		if _, ok := llrLoc[doc]; !ok {
			llrLoc[doc] = make(map[string]*code.Code)
		}
		return llrLoc[doc][symbol]
	}

	setLlrLocForSymbolInDocument := func(doc, symbol string, loc *code.Code) {
		if _, ok := llrLoc[doc]; !ok {
			llrLoc[doc] = make(map[string]*code.Code)
		}
		llrLoc[doc][symbol] = loc
	}

	// linksMatch compares an array of requirement IDs with the given array of ReqLink's and matches
	// each element if the requirement ID matches (ignoring other members of the ReqLink struct).
	linksMatch := func(lhs []string, rhs []code.ReqLink) bool {
		if len(lhs) != len(rhs) {
			return false
		}

		rhsMatched := make([]bool, len(rhs))

		for lhs_idx := range lhs {
			found := false
			for rhs_idx := range rhs {
				if !rhsMatched[rhs_idx] && lhs[lhs_idx] == rhs[rhs_idx].Id {
					rhsMatched[rhs_idx] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	// Walk the code tags, resolving links and looking for errors
	for _, tags := range rg.CodeTags {
		for _, code := range tags {
			links := getParentIdsForSymbolInDocument(code.Document.Path, code.Symbol)

			if len(code.Links) == 0 {
				continue
			}

			if code.Symbol == "" {
				continue
			}

			if len(links) == 0 {
				setParentIdsForSymbolInDocument(code.Document.Path, code.Symbol, code.Links)
				setLlrLocForSymbolInDocument(code.Document.Path, code.Symbol, code)
				continue
			}

			prevLoc := getLlrLocForSymbolInDocument(code.Document.Path, code.Symbol)

			if !linksMatch(links, code.Links) {
				var errorMessage error
				if prevLoc.CodeFile.Path > code.CodeFile.Path || (prevLoc.CodeFile.Path == code.CodeFile.Path && prevLoc.Line >= code.Line) {
					errorMessage = fmt.Errorf("LLR declarations differ in %s@%s:%d and %s@%s:%d.",
						prevLoc.Tag, prevLoc.CodeFile.Path, prevLoc.Line, code.Tag, code.CodeFile.Path, code.Line)
				} else {
					errorMessage = fmt.Errorf("LLR declarations differ in %s@%s:%d and %s@%s:%d.",
						code.Tag, code.CodeFile.Path, code.Line, prevLoc.Tag, prevLoc.CodeFile.Path, prevLoc.Line)
				}
				issue := diagnostics.Issue{
					Line:     code.Line,
					Path:     code.CodeFile.Path,
					RepoName: code.CodeFile.RepoName,
					Error:    errorMessage,
					Severity: diagnostics.IssueSeverityMajor,
					Type:     diagnostics.IssueTypeInvalidRequirementInCode,
				}
				issues = append(issues, issue)
			}
		}
	}
	return issues, getParentIdsForSymbolInDocument
}

// TODO(ja): Make this more modular and resolve diagnostics at multiple levels (we already know some of these diagnostics just by parsing code)
// Resolve walks the requirements graph and resolves the links between different levels of requirements
// and with code tags. References to requirements within requirements text is checked as well as validity
// of attributes against the schema for their document. Any errors encountered such as links to
// non-existent requirements are returned in a list of issues.
// @llr REQ-TRAQ-SWL-10, REQ-TRAQ-SWL-11, REQ-TRAQ-SWL-67, REQ-TRAQ-SWL-69
func (rg *ReqGraph) Resolve() []diagnostics.Issue {
	issues := make([]diagnostics.Issue, 0)

	// Walk the requirements, resolving links and looking for errors
	for _, req := range rg.Reqs {
		if req.IsDeleted() {
			continue
		}

		// Validate requirement Id
		if !req.Document.Schema.Requirements.MatchString(req.ID) {
			issue := diagnostics.Issue{
				Line:     req.Position,
				Path:     req.Document.Path,
				RepoName: req.RepoName,
				Error:    fmt.Errorf("Requirement `%s` in document `%s` does not match required regexp `%s`", req.ID, req.Document.Path, req.Document.Schema.Requirements),
				Severity: diagnostics.IssueSeverityMajor,
				Type:     diagnostics.IssueTypeInvalidRequirementId,
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
					issue := diagnostics.Issue{
						Line:     req.Position,
						Path:     req.Document.Path,
						RepoName: req.RepoName,
						Error:    errors.New("Invalid parent of requirement " + req.ID + ": " + parentID + " is deleted."),
						Severity: diagnostics.IssueSeverityMajor,
						Type:     diagnostics.IssueTypeInvalidParent,
					}
					issues = append(issues, issue)
				}
				if req.Variant == ReqVariantRequirement {
					if err := req.validateLink(parent); err != nil {
						issue := diagnostics.Issue{
							Line:     req.Position,
							Path:     req.Document.Path,
							RepoName: req.RepoName,
							Error:    err,
							Severity: diagnostics.IssueSeverityMajor,
							Type:     diagnostics.IssueTypeInvalidParent,
						}
						issues = append(issues, issue)
					}
				}
				parent.Children = append(parent.Children, req)
				req.Parents = append(req.Parents, parent)
			} else {
				issue := diagnostics.Issue{
					Line:     req.Position,
					Path:     req.Document.Path,
					RepoName: req.RepoName,
					Error:    errors.New("Invalid parent of requirement " + req.ID + ": " + parentID + " does not exist."),
					Severity: diagnostics.IssueSeverityMajor,
					Type:     diagnostics.IssueTypeInvalidParent,
				}
				issues = append(issues, issue)
			}
		}
		// Validate references to requirements in body text
		matches := reReqID.FindAllStringSubmatchIndex(req.Body, -1)
		for _, ids := range matches {
			reqID := req.Body[ids[0]:ids[1]]
			v, reqFound := rg.Reqs[reqID]
			if !reqFound {
				issue := diagnostics.Issue{
					Line:     req.Position,
					Path:     req.Document.Path,
					RepoName: req.RepoName,
					Error:    fmt.Errorf("Invalid reference to non existent requirement %s in body of %s.", reqID, req.ID),
					Severity: diagnostics.IssueSeverityMajor,
					Type:     diagnostics.IssueTypeInvalidRequirementReference,
				}
				issues = append(issues, issue)
			} else if v.IsDeleted() {
				issue := diagnostics.Issue{
					Line:     req.Position,
					Path:     req.Document.Path,
					RepoName: req.RepoName,
					Error:    fmt.Errorf("Invalid reference to deleted requirement %s in body of %s.", reqID, req.ID),
					Severity: diagnostics.IssueSeverityMajor,
					Type:     diagnostics.IssueTypeInvalidRequirementReference,
				}
				issues = append(issues, issue)
			}
		}

		// TODO check for references to missing or deleted requirements in attribute text
	}

	symbolIssues, getParentIdsForSymbolInDocument := rg.deduplicateCodeSymbols()
	issues = append(issues, symbolIssues...)

	for _, tags := range rg.CodeTags {
		for _, code := range tags {
			parentIds := []string{}
			if code.Symbol != "" {
				parentIds = getParentIdsForSymbolInDocument(code.Document.Path, code.Symbol)
			} else {
				for _, link := range code.Links {
					parentIds = append(parentIds, link.Id)
				}
			}

			if len(parentIds) == 0 && !code.Optional {
				issue := diagnostics.Issue{
					Line:     code.Line,
					Path:     code.CodeFile.Path,
					RepoName: code.CodeFile.RepoName,
					Error:    fmt.Errorf("Function %s@%s:%d has no parents.", code.Tag, code.CodeFile.String(), code.Line),
					Severity: diagnostics.IssueSeverityMajor,
					Type:     diagnostics.IssueTypeMissingRequirementInCode,
				}
				issues = append(issues, issue)
			}
			for _, parentID := range parentIds {
				if !code.Document.Schema.Requirements.MatchString(parentID) {
					issue := diagnostics.Issue{
						Line:     code.Line,
						Path:     code.CodeFile.Path,
						RepoName: code.CodeFile.RepoName,
						Error: fmt.Errorf("Invalid reference in function %s@%s:%d in repo `%s`, `%s` does not match requirement format in document `%s`.",
							code.Tag, code.CodeFile.Path, code.Line, code.CodeFile.RepoName, parentID, code.Document.Path),
						Severity: diagnostics.IssueSeverityMajor,
						Type:     diagnostics.IssueTypeInvalidRequirementInCode,
					}
					issues = append(issues, issue)
				}

				parent := rg.Reqs[parentID]
				if parent != nil {
					if parent.IsDeleted() {
						issue := diagnostics.Issue{
							Line:     code.Line,
							Path:     code.CodeFile.Path,
							RepoName: code.CodeFile.RepoName,
							Error: fmt.Errorf("Invalid reference in function %s@%s:%d in repo `%s`, %s is deleted.",
								code.Tag, code.CodeFile.Path, code.Line, code.CodeFile.RepoName, parentID),
							Severity: diagnostics.IssueSeverityMajor,
							Type:     diagnostics.IssueTypeInvalidRequirementInCode,
						}
						issues = append(issues, issue)
					}

					parent.Tags = append(parent.Tags, code)
				} else {
					issue := diagnostics.Issue{
						Line:     code.Line,
						Path:     code.CodeFile.Path,
						RepoName: code.CodeFile.RepoName,
						Error: fmt.Errorf("Invalid reference in function %s@%s:%d in repo `%s`, %s does not exist.",
							code.Tag, code.CodeFile.Path, code.Line, code.CodeFile.RepoName, parentID),
						Severity: diagnostics.IssueSeverityMajor,
						Type:     diagnostics.IssueTypeInvalidRequirementInCode,
					}
					issues = append(issues, issue)
				}
			}
		}
	}

	// Walk through the requirements one last time to ensure that if they are tested they are also implemented.
	// We need to do it at this point, since now the links to the Tags are all set
	for _, req := range rg.Reqs {
		if !req.Document.HasImplementation() || req.IsDeleted() {
			continue
		}

		implemented := false
		tested := false
		for _, tag := range req.Tags {
			if tag.CodeFile.Type.Matches(code.CodeTypeImplementation) {
				implemented = true
			}
			if tag.CodeFile.Type.Matches(code.CodeTypeTests) {
				tested = true
			}
			if implemented && tested {
				break
			}
		}

		if !implemented {
			if tested {
				issue := diagnostics.Issue{
					Line:     req.Position,
					Path:     req.Document.Path,
					RepoName: req.RepoName,
					Error:    fmt.Errorf("Requirement %s is tested, but it is not implemented.", req.ID),
					Severity: diagnostics.IssueSeverityMajor,
					Type:     diagnostics.IssueTypeReqTestedButNotImplemented,
				}
				issues = append(issues, issue)
			} else {
				issue := diagnostics.Issue{
					Line:     req.Position,
					Path:     req.Document.Path,
					RepoName: req.RepoName,
					Error:    fmt.Errorf("Requirement %s is not implemented.", req.ID),
					Severity: diagnostics.IssueSeverityNote,
					Type:     diagnostics.IssueTypeReqNotImplemented,
				}
				issues = append(issues, issue)

			}
		} else if !tested {
			issue := diagnostics.Issue{
				Line:     req.Position,
				Path:     req.Document.Path,
				RepoName: req.RepoName,
				Error:    fmt.Errorf("Requirement %s is not tested.", req.ID),
				Severity: diagnostics.IssueSeverityNote,
				Type:     diagnostics.IssueTypeReqNotTested,
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
func (r *Req) checkAttributes() []diagnostics.Issue {
	var schemaAttributes map[string]*config.Attribute
	switch r.Variant {
	case ReqVariantRequirement:
		schemaAttributes = r.Document.Schema.Attributes
	case ReqVariantAssumption:
		schemaAttributes = r.Document.Schema.AsmAttributes
	}

	var issues []diagnostics.Issue
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
			issue := diagnostics.Issue{
				Line:     r.Position,
				Path:     r.Document.Path,
				RepoName: r.RepoName,
				Error:    fmt.Errorf("Requirement '%s' is missing attribute '%s'.", r.ID, name),
				Severity: diagnostics.IssueSeverityMajor,
				Type:     diagnostics.IssueTypeMissingAttribute,
			}
			issues = append(issues, issue)
		} else if reqValuePresent {
			if attribute.Type == config.AttributeAny {
				anyCount++
			}

			if !attribute.Value.MatchString(reqValue) {
				issue := diagnostics.Issue{
					Line:     r.Position,
					Path:     r.Document.Path,
					RepoName: r.RepoName,
					Error:    fmt.Errorf("Requirement '%s' has invalid value '%s' in attribute '%s'.", r.ID, reqValue, name),
					Severity: diagnostics.IssueSeverityMajor,
					Type:     diagnostics.IssueTypeInvalidAttributeValue,
				}
				issues = append(issues, issue)
			}
		}
	}

	if len(anyAttributes) > 0 && anyCount == 0 {
		sort.Strings(anyAttributes)
		issue := diagnostics.Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Requirement '%s' is missing at least one of the attributes '%s'.", r.ID, strings.Join(anyAttributes, ",")),
			Severity: diagnostics.IssueSeverityMajor,
			Type:     diagnostics.IssueTypeMissingAttribute,
		}
		issues = append(issues, issue)
	}

	// Iterate the requirement attributes to check for unknown ones
	for name := range r.Attributes {
		if _, present := schemaAttributes[strings.ToUpper(name)]; !present {
			issue := diagnostics.Issue{
				Line:     r.Position,
				Path:     r.Document.Path,
				RepoName: r.RepoName,
				Error:    fmt.Errorf("Requirement '%s' has unknown attribute '%s'.", r.ID, name),
				Severity: diagnostics.IssueSeverityMajor,
				Type:     diagnostics.IssueTypeUnknownAttribute,
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// validateLink iterates through the link options for the requirement and checks if the parent ID is valid
// @llr REQ-TRAQ-SWL-76
func (r *Req) validateLink(parent *Req) error {
	for _, link := range r.Document.LinkSpecs {
		if !link.Child.Re.MatchString(r.ID) {
			// link option doesn't apply to this requirement
			continue
		}

		if link.Child.AttrKey != "" {
			value, present := r.Attributes[link.Child.AttrKey]
			if !present || !link.Child.AttrVal.MatchString(value) {
				// link option doesn't apply to this requirement
				continue
			}
		}

		if !link.Parent.Re.MatchString(parent.ID) {
			// link option doesn't apply to this parent
			continue
		}

		if link.Parent.AttrKey != "" {
			value, present := parent.Attributes[link.Parent.AttrKey]
			if !present || !link.Parent.AttrVal.MatchString(value) {
				return fmt.Errorf("Requirement '%s' has invalid parent link ID '%s' with attribute value '%s'=='%s'.", r.ID, parent.ID, link.Parent.AttrKey, value)
			}
		}

		return nil
	}

	return fmt.Errorf("Requirement '%s' has invalid parent link ID '%s'.", r.ID, parent.ID)
}

// checkID verifies that the requirement is not duplicated
// @llr REQ-TRAQ-SWL-25, REQ-TRAQ-SWL-26, REQ-TRAQ-SWL-28
func (r *Req) checkID(document *config.Document, expectedIDNumber int, isReqPresent []bool) []diagnostics.Issue {
	var issues []diagnostics.Issue
	reqIDComps := strings.Split(r.ID, "-") // results in an array such as [REQ PROJECT REQTYPE 1234]
	// check requirement name, no need to check prefix because it would not have been parsed otherwise
	if reqIDComps[1] != string(document.ReqSpec.Prefix) {
		issue := diagnostics.Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Incorrect project abbreviation for requirement %s. Expected %s, got %s.", r.ID, document.ReqSpec.Prefix, reqIDComps[1]),
			Severity: diagnostics.IssueSeverityMajor,
			Type:     diagnostics.IssueTypeInvalidRequirementId,
		}
		issues = append(issues, issue)
	}
	if reqIDComps[2] != string(document.ReqSpec.Level) {
		issue := diagnostics.Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Incorrect requirement type for requirement %s. Expected %s, got %s.", r.ID, document.ReqSpec.Level, reqIDComps[2]),
			Severity: diagnostics.IssueSeverityMajor,
			Type:     diagnostics.IssueTypeInvalidRequirementId,
		}
		issues = append(issues, issue)
	}
	if reqIDComps[3][0] == '0' {
		issue := diagnostics.Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Requirement number cannot begin with a 0: %s. Got %s.", r.ID, reqIDComps[3]),
			Severity: diagnostics.IssueSeverityMajor,
			Type:     diagnostics.IssueTypeInvalidRequirementId,
		}
		issues = append(issues, issue)
	}

	currentID, err2 := strconv.Atoi(reqIDComps[3])
	if err2 != nil {
		issue := diagnostics.Issue{
			Line:     r.Position,
			Path:     r.Document.Path,
			RepoName: r.RepoName,
			Error:    fmt.Errorf("Invalid requirement sequence number for %s (failed to parse): %s", r.ID, reqIDComps[3]),
			Severity: diagnostics.IssueSeverityMajor,
			Type:     diagnostics.IssueTypeInvalidRequirementId,
		}
		issues = append(issues, issue)
	} else {
		if currentID < 1 {
			issue := diagnostics.Issue{
				Line:     r.Position,
				Path:     r.Document.Path,
				RepoName: r.RepoName,
				Error:    fmt.Errorf("Invalid requirement sequence number for %s: first requirement has to start with 001.", r.ID),
				Severity: diagnostics.IssueSeverityMajor,
				Type:     diagnostics.IssueTypeInvalidRequirementId,
			}
			issues = append(issues, issue)
		} else {
			if isReqPresent[currentID-1] {
				issue := diagnostics.Issue{
					Line:     r.Position,
					Path:     r.Document.Path,
					RepoName: r.RepoName,
					Error:    fmt.Errorf("Invalid requirement sequence number for %s, is duplicate.", r.ID),
					Severity: diagnostics.IssueSeverityMajor,
					Type:     diagnostics.IssueTypeInvalidRequirementId,
				}
				issues = append(issues, issue)
			} else {
				if currentID != expectedIDNumber {
					issue := diagnostics.Issue{
						Line:     r.Position,
						Path:     r.Document.Path,
						RepoName: r.RepoName,
						Error:    fmt.Errorf("Invalid requirement sequence number for %s: missing requirements in between. Expected ID Number %d.", r.ID, expectedIDNumber),
						Severity: diagnostics.IssueSeverityMajor,
						Type:     diagnostics.IssueTypeInvalidRequirementId,
					}
					issues = append(issues, issue)
				}
			}
			isReqPresent[currentID-1] = true
		}
	}

	return issues
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

// CreateFilter reads the filter regular expressions from the command line arguments and
// compiles them into a filter structure ready to use
// @llr REQ-TRAQ-SWL-19, REQ-TRAQ-SWL-73
func CreateFilter(idFilter, titleFilter, bodyFilter string, attributeFilter []string) (ReqFilter, error) {
	filter := ReqFilter{} // Filter for report generation
	filter.AttributeRegexp = make(map[string]*regexp.Regexp, 0)
	var err error
	if len(idFilter) > 0 {
		filter.IDRegexp, err = regexp.Compile(idFilter)
		if err != nil {
			return filter, err
		}
	}
	if len(titleFilter) > 0 {
		filter.TitleRegexp, err = regexp.Compile(titleFilter)
		if err != nil {
			return filter, err
		}
	}
	if len(bodyFilter) > 0 {
		filter.BodyRegexp, err = regexp.Compile(bodyFilter)
		if err != nil {
			return filter, err
		}
	}
	if len(attributeFilter) > 0 {
		for _, f := range attributeFilter {
			if strings.Contains(f, "=") {
				parts := strings.Split(f, "=")
				r, err := regexp.Compile(parts[1])
				if err != nil {
					return filter, err
				}
				filter.AttributeRegexp[strings.ToUpper(parts[0])] = r
			} else {
				if filter.AnyAttributeRegexp != nil {
					return filter, errors.New("cannot specify more than one any attribute filter")
				}
				filter.AnyAttributeRegexp, err = regexp.Compile(f)
				if err != nil {
					return filter, err
				}
			}
		}
	}
	return filter, nil
}

// IsEmpty returns whether the filter has no restriction.
// @llr REQ-TRAQ-SWL-20, REQ-TRAQ-SWL-21
func (f ReqFilter) IsEmpty() bool {
	return f.IDRegexp == nil && f.TitleRegexp == nil &&
		f.BodyRegexp == nil && f.AnyAttributeRegexp == nil &&
		len(f.AttributeRegexp) == 0
}

// Matches returns true if the requirement matches the filter AND its ID is in the diffs map, if any.
// @llr REQ-TRAQ-SWL-19, REQ-TRAQ-SWL-73
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
