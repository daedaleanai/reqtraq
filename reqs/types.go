/*
 * Types used for requirements
 */
package reqs

import (
	"regexp"

	"github.com/daedaleanai/reqtraq/code"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/diagnostics"
	"github.com/daedaleanai/reqtraq/repos"
)

// ReqGraph holds the complete information about a set of requirements and associated code tags.
type ReqGraph struct {
	// Reqs contains the requirements by ID.
	Reqs map[string]*Req
	// CodeTags contains the source code functions per file.
	// The keys are paths relative to the git repo path.
	CodeTags map[code.CodeFile][]*code.Code
	// Issues which have been found while analyzing the graph.
	// This is extended in multiple places.
	Issues []diagnostics.Issue
	// Holds configuration of reqtraq for all associated repositories
	ReqtraqConfig *config.Config
}

// Represents the type of requirement (assumption or requirement)
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
	Tags  []*code.Code
	Title string
	Body  string
	// Attributes of the requirement by uppercase name.
	Attributes map[string]string
	Position   int
	// Link back to the document where the requirement is defined and the name of the repository
	Document *config.Document
	RepoName repos.RepoName
}

// ReqFilter holds the different parameters used to filter the requirements set.
type ReqFilter struct {
	IDRegexp           *regexp.Regexp
	TitleRegexp        *regexp.Regexp
	BodyRegexp         *regexp.Regexp
	AnyAttributeRegexp *regexp.Regexp
	AttributeRegexp    map[string]*regexp.Regexp
}

// ReqFormatType defines what type of requirement we are parsing. None, a heading based requirement or a table of
// requirements.
type ReqFormatType int

const (
	None ReqFormatType = iota
	Heading
	Table
)
