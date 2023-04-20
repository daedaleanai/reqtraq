package diagnostics

import "github.com/daedaleanai/reqtraq/repos"

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
	IssueTypeReqNotImplemented
	IssueTypeReqNotTested
)

type IssueSeverity uint

const (
	IssueSeverityMajor IssueSeverity = iota
	IssueSeverityMinor
	IssueSeverityNote
)

type Issue struct {
	RepoName repos.RepoName
	Path     string
	Line     int
	Error    error
	Severity IssueSeverity
	Type     IssueType
}