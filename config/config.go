// Reads configuration data from a reqtraq_config.json file

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
)

type ReqLevel string
type ReqPrefix string
type Arch string

/// Internal types for parsing json files

type jsonRepoLink struct {
	RepoName   repos.RepoName   `json:"repoName"`
	RemotePath repos.RemotePath `json:"repoUrl"`
}

type jsonAttribute struct {
	Name     string `json:"name"`
	Required string `json:"required"`
	Value    string `json:"value"`
}

type jsonFileQueryBase struct {
	Paths           []string `json:"paths"`
	MatchingPattern string   `json:"matchingPattern"`
	IgnoredPatterns []string `json:"ignoredPatterns"`
}

type jsonFileQuery struct {
	jsonFileQueryBase
	ArchPatterns map[Arch]jsonFileQueryBase `json:"archPatterns"`
}

type jsonArchCompilerData struct {
	CompilationDatabase string   `json:"compilationDatabase"`
	CompilerArguments   []string `json:"compilerArguments"`
}

type jsonImplementation struct {
	Archs               map[Arch]jsonArchCompilerData `json:"archs"`
	Code                jsonFileQuery                 `json:"code"`
	Tests               jsonFileQuery                 `json:"tests"`
	CodeParser          string                        `json:"codeParser"`
	CompilationDatabase string                        `json:"compilationDatabase"`
	CompilerArguments   []string                      `json:"compilerArguments"`
}

type jsonParent struct {
	Prefix          ReqPrefix     `json:"prefix"`
	Level           ReqLevel      `json:"level"`
	ParentAttribute jsonAttribute `json:"parentAttribute"`
	ChildAttribute  jsonAttribute `json:"childAttribute"`
}

type jsonParents []jsonParent

type jsonImplementations []jsonImplementation

type jsonDoc struct {
	Path           string              `json:"path"`
	Prefix         ReqPrefix           `json:"prefix"`
	Level          ReqLevel            `json:"level"`
	Parent         jsonParents         `json:"parent"`
	Attributes     []jsonAttribute     `json:"attributes"`
	AsmAttributes  []jsonAttribute     `json:"asmAttributes"`
	Implementation jsonImplementations `json:"implementation"`
}

type jsonConfig struct {
	RepoName         repos.RepoName  `json:"repoName"`
	CommonAttributes []jsonAttribute `json:"commonAttributes"`
	ParentRepo       jsonRepoLink    `json:"parentRepository"`
	ChildrenRepos    []jsonRepoLink  `json:"childrenRepositories"`
	Docs             []jsonDoc       `json:"documents"`
}

/// Types exported for application use

// A type of attribute for a requirement
type AttributeType uint

// The enumeration of possible attribute types
const (
	// The attribute must always be present in the requirement
	AttributeRequired AttributeType = iota
	// The attribute can be optionally present in the requirement
	AttributeOptional
	// At least one of the attributes with type any must be present in the requirement
	AttributeAny
)

// An structure defining an attribute with the given type and value. The attribute must match the
// regular expression in value to be valid
type Attribute struct {
	Type  AttributeType
	Value *regexp.Regexp
}

// A structure describing the implementation for a given certification document,
// for architecture-dependent codebases
type ArchImplementation struct {
	CodeFiles           []string
	TestFiles           []string
	CompilationDatabase string
	CompilerArguments   []string
}

// A structure describing the implementation for a given certification document.
type Implementation struct {
	ArchImplementation
	CodeParser string
	Archs      map[Arch]ArchImplementation
}

// The schema for requirements inside a certification document
type Schema struct {
	Requirements  *regexp.Regexp
	Attributes    map[string]*Attribute
	AsmAttributes map[string]*Attribute
}

// A requirement specification. Identifies the form of requirements in a document
type ReqSpec struct {
	Prefix  ReqPrefix
	Level   ReqLevel
	Re      *regexp.Regexp
	AttrKey string
	AttrVal *regexp.Regexp
}

// A link specification. Identifies valid child and parent ends of a link.
type LinkSpec struct {
	Child  ReqSpec
	Parent ReqSpec
}

// A certification document with its given requirement specification and schema, as well as its
// implementation in terms of code and its location in the repository
type Document struct {
	Path           string
	ReqSpec        ReqSpec
	LinkSpecs      []LinkSpec
	Schema         Schema
	Implementation []Implementation
}

// A configuration for a single repository, which is made of documents.
type RepoConfig struct {
	Documents []Document
}

// A global configuration structure for a repo, its parents and its children.
type Config struct {
	TargetRepo repos.RepoName
	Repos      map[repos.RepoName]RepoConfig
}

// Selects whether all children of the parent repositories should be traversed as part of the
// configuration or only parents are traversed
var DirectDependenciesOnly bool = false

// Top level function to parse the configuration file from the given path in the current repository
// @llr REQ-TRAQ-SWL-53
func ParseConfig(repoPath repos.RepoPath) (Config, error) {
	jsonConfig, err := readJsonConfigFromRepo(repoPath)
	if err != nil {
		return Config{}, errors.Wrapf(err, "The requested config path `%s` does not contain a valid repository", repoPath)
	}

	config := Config{
		TargetRepo: jsonConfig.RepoName,
		Repos:      make(map[repos.RepoName]RepoConfig),
	}

	commonAttributes := make(map[string]*Attribute)

	err = config.parseConfigFile(jsonConfig, &commonAttributes)
	if err != nil {
		return Config{}, err
	}

	config.appendCommonAttributes(&commonAttributes)

	return config, nil
}

// Returns true if the document has associated implementation
// @llr REQ-TRAQ-SWL-56
func (doc *Document) HasImplementation() bool {
	var hasImpl bool
	for _, impl := range doc.Implementation {
		if len(impl.CodeFiles) != 0 {
			hasImpl = true
		}
	}
	return hasImpl
}

// Returns true if the document matches the given requirement spec.
// @llr REQ-TRAQ-SWL-55
func (doc *Document) MatchesSpec(reqSpec ReqSpec) bool {
	return (reqSpec.Level == doc.ReqSpec.Level) && (reqSpec.Prefix == doc.ReqSpec.Prefix)
}

// Converts the requirement specification to a REQ string
// @llr REQ-TRAQ-SWL-55
func (req ReqSpec) String() string {
	if req.AttrKey == "" {
		return fmt.Sprintf("REQ-%s-%s", req.Prefix, req.Level)
	} else {
		return fmt.Sprintf("REQ-%s-%s (%s == %s)", req.Prefix, req.Level, req.AttrKey, strings.Trim(req.AttrVal.String(), "^$"))
	}
}

// Finds the associated Document for a certdoc located at the given path or a nil document if it
// is not found
// @llr REQ-TRAQ-SWL-54
func (config *Config) FindCertdoc(path string) (repos.RepoName, *Document) {
	for repoName := range config.Repos {
		for docIdx := range config.Repos[repoName].Documents {
			if filepath.Base(config.Repos[repoName].Documents[docIdx].Path) == filepath.Base(path) {
				return repoName, &config.Repos[repoName].Documents[docIdx]
			}
		}
	}
	return "", nil
}

// Builds a map of the linked child -> parent requirement specification to know what specs are related by a
// parent/children relationship
// @llr REQ-TRAQ-SWL-54
func (config *Config) GetLinkedSpecs() []LinkSpec {
	var links []LinkSpec

	for repoName := range config.Repos {
		for docIdx := range config.Repos[repoName].Documents {
			doc := &config.Repos[repoName].Documents[docIdx]
			for _, link := range doc.LinkSpecs {
				if link.Child.Level != "" && link.Child.Prefix != "" {
					links = append(links, link)
				}
			}
		}
	}

	return links
}

// Loads the information for the base repository from git
// @llr REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-81
func LoadBaseRepoInfo(repoPath string) {
	// See details about "working directory" in https://git-scm.com/docs/githooks
	bare, err := linepipes.Single(linepipes.Run("git", "-C", repoPath, "rev-parse", "--is-bare-repository"))
	if err != nil {
		log.Fatalf("Failed to check Git repository type. Are you running reqtraq in a Git repo?\n%s", err)
	}
	if bare == "true" {
		log.Fatal("Reqtraq cannot be used in bare checkouts")
	}

	// Get the absolute path to the repo.
	toplevel, err := linepipes.Single(linepipes.Run("git", "-C", repoPath, "rev-parse", "--show-toplevel"))
	if err != nil {
		log.Fatal(err)
	}

	basePath := repos.RepoPath(toplevel)

	config, err := readJsonConfigFromRepo(basePath)
	if err != nil {
		log.Fatalf("Error reading configuration in path `%s`: %v", basePath, err)
	}

	repos.SetBaseRepoInfo(basePath, config.RepoName)
}

// Reads a json configuration file from the specified repository path.
// The file is always located at reqtraq_config.json
// @llr REQ-TRAQ-SWL-53
func readJsonConfigFromRepo(repoPath repos.RepoPath) (jsonConfig, error) {
	// Read parent config and parse that
	configPath := filepath.Join(string(repoPath), "reqtraq_config.json")

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return jsonConfig{}, fmt.Errorf("Error opening configuration file: %s", configPath)
	}

	var config jsonConfig
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return jsonConfig{}, errors.Wrapf(err, "Error while parsing configuration file `%s`", configPath)
	}
	return config, nil
}

// UnmarshalJSON implements the Unmarshaler interface for the jsonParents type, allowing the 'parent' field to be a single
// struct or array of structs when defined in the config file.
// @llr REQ-TRAQ-SWL-53
func (op *jsonParents) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("no bytes to unmarshal")
	}

	*op = jsonParents{}

	// Use the first character to determine the type
	switch data[0] {
	case '{':
		var parent jsonParent
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&parent); err != nil {
			return err
		}
		*op = append(*op, parent)
		return nil

	case '[':
		// we can't use the jsonParents type or Unmarshal will call back this function
		var parents []jsonParent
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&parents); err != nil {
			return err
		}
		*op = append(*op, parents...)
		return nil
	}

	return fmt.Errorf("malformed JSON, expected '[' or '{' in parents field, got %c", data[0])
}

// UnmarshalJSON implements the Unmarshaler interface for the jsonImplementations type, allowing the
// 'Implementation' field to be a single struct or array of structs when defined in the config file.
// @llr REQ-TRAQ-SWL-87
func (impls *jsonImplementations) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("no bytes to unmarshal")
	}

	*impls = jsonImplementations{}

	var multiImpls []jsonImplementation
	{
		decoder := json.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&multiImpls); err == nil {
			*impls = jsonImplementations(multiImpls)
			return nil
		}
	}

	// Fall back to the legacy format and try again
	var legacyImpl jsonImplementation
	{
		decoder := json.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&legacyImpl); err == nil {
			*impls = jsonImplementations([]jsonImplementation{legacyImpl})
			return nil
		}
	}

	return fmt.Errorf("malformed JSON, Unable to parse jsonImplementations")
}

// Parses an a single attribute from its json description
// @llr REQ-TRAQ-SWL-53
func parseAttribute(rawAttribute jsonAttribute) (string, Attribute, error) {
	var attribute Attribute
	switch rawAttribute.Required {
	case "true":
		attribute.Type = AttributeRequired
	case "any":
		attribute.Type = AttributeAny
	case "false":
		attribute.Type = AttributeOptional
	case "":
		attribute.Type = AttributeRequired
	default:
		return "", Attribute{}, fmt.Errorf("Unable to parse attribute `required` field: `%s`", rawAttribute.Required)
	}

	if rawAttribute.Value == "" {
		attribute.Value = regexp.MustCompile(".*")
	} else {
		var err error
		attribute.Value, err = regexp.Compile(rawAttribute.Value)
		if err != nil {
			return "", Attribute{}, err
		}
	}

	return strings.ToUpper(rawAttribute.Name), attribute, nil
}

// parseParent creates a link specification from a json description
// @llr REQ-TRAQ-SWL-53
func parseParent(rawParent jsonParent, childPrefix ReqPrefix, childLevel ReqLevel) (LinkSpec, error) {
	newLink := LinkSpec{}

	newLink.Child.Prefix = childPrefix
	newLink.Child.Level = childLevel
	newLink.Child.Re = regexp.MustCompile(fmt.Sprintf("REQ-%s-%s-(\\d+)", childPrefix, childLevel))

	childName, childAttr, err := parseAttribute(rawParent.ChildAttribute)
	if err != nil {
		return newLink, err
	}
	newLink.Child.AttrKey = childName
	newLink.Child.AttrVal = childAttr.Value

	newLink.Parent.Prefix = rawParent.Prefix
	newLink.Parent.Level = rawParent.Level
	newLink.Parent.Re = regexp.MustCompile(fmt.Sprintf("REQ-%s-%s-(\\d+)", rawParent.Prefix, rawParent.Level))

	parentName, parentAttr, err := parseAttribute(rawParent.ParentAttribute)
	if err != nil {
		return newLink, err
	}
	newLink.Parent.AttrKey = parentName
	newLink.Parent.AttrVal = parentAttr.Value

	return newLink, nil
}

// Finds all matching files for the given query under the given repository.
// @llr REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-56
func (fileQuery *jsonFileQuery) findAllMatchingFiles(repoName repos.RepoName, arch ...Arch) ([]string, error) {
	var queryMatchingPattern string
	var queryIgnoredPatterns []string
	var queryPaths []string

	if len(arch) == 0 {
		queryMatchingPattern = fileQuery.MatchingPattern
		queryIgnoredPatterns = fileQuery.IgnoredPatterns
		queryPaths = fileQuery.Paths
	} else {
		_, ok := fileQuery.ArchPatterns[arch[0]]
		if !ok {
			return []string{}, nil
		}

		queryMatchingPattern = fileQuery.ArchPatterns[arch[0]].MatchingPattern
		queryIgnoredPatterns = fileQuery.ArchPatterns[arch[0]].IgnoredPatterns
		queryPaths = fileQuery.ArchPatterns[arch[0]].Paths
	}

	var matchingPattern *regexp.Regexp = nil
	if queryMatchingPattern != "" {
		var err error
		matchingPattern, err = regexp.Compile(queryMatchingPattern)
		if err != nil {
			return []string{}, err
		}
	}

	var collectedFiles = []string{}

	var ignoredPatterns []*regexp.Regexp
	for _, pattern := range queryIgnoredPatterns {
		compiledPattern, err := regexp.Compile(pattern)
		if err != nil {
			return []string{}, fmt.Errorf("Unable to parse `%s` as a regular expression", pattern)
		}
		ignoredPatterns = append(ignoredPatterns, compiledPattern)
	}

	for _, path := range queryPaths {
		matched_files, err := repos.FindFilesInDirectory(repoName, path, matchingPattern, ignoredPatterns)
		if err != nil {
			return []string{}, err
		}
		collectedFiles = append(collectedFiles, matched_files...)
	}

	return collectedFiles, nil
}

// Parses an implementation of a document, returning it or an error if the parsing failed
// @llr REQ-TRAQ-SWL-56, REQ-TRAQ-SWL-64, REQ-TRAQ-SWL-87
func parseImplementation(repoName repos.RepoName, impl *jsonImplementation) (*Implementation, error) {
	parsedImpl := Implementation{
		Archs: map[Arch]ArchImplementation{},
	}

	// Trigger a warning for every arch with matching rules
	// that has no compilation database or arguments defined
	for arch := range impl.Code.ArchPatterns {
		var exists bool
		_, exists = impl.Archs[arch]
		if !exists {
			fmt.Printf("Warning: %q has matching rules for code, but it is not mentioned in the top level `archs` field, so it will not actually be used for matching its files.\n", arch)
		}
	}

	for arch := range impl.Tests.ArchPatterns {
		var exists bool
		_, exists = impl.Archs[arch]
		if !exists {
			fmt.Printf("Warning: %q has matching rules for tests, but it is not mentioned in the top level `archs` field, so it will not actually be used for matching its files.\n", arch)
		}
	}

	for arch := range impl.Archs {
		var newArchEntry ArchImplementation
		newArchEntry.CompilationDatabase = impl.Archs[arch].CompilationDatabase
		newArchEntry.CompilerArguments = impl.Archs[arch].CompilerArguments

		codeFiles, err := impl.Code.findAllMatchingFiles(repoName, arch)
		newArchEntry.CodeFiles = codeFiles
		if err != nil {
			return nil, err
		}

		testFiles, err := impl.Tests.findAllMatchingFiles(repoName, arch)
		newArchEntry.TestFiles = testFiles
		if err != nil {
			return nil, err
		}

		parsedImpl.Archs[arch] = newArchEntry
	}

	codeFiles, err := impl.Code.findAllMatchingFiles(repoName)
	parsedImpl.CodeFiles = codeFiles
	if err != nil {
		return nil, err
	}

	testFiles, err := impl.Tests.findAllMatchingFiles(repoName)
	parsedImpl.TestFiles = testFiles
	if err != nil {
		return nil, err
	}
	parsedImpl.CompilationDatabase = impl.CompilationDatabase
	parsedImpl.CompilerArguments = impl.CompilerArguments
	if parsedImpl.CompilerArguments == nil {
		parsedImpl.CompilerArguments = []string{}
	}
	parsedImpl.CodeParser = impl.CodeParser
	if parsedImpl.CodeParser == "" {
		// Default to ctags parser, which is always built-in
		parsedImpl.CodeParser = "ctags"
	}
	return &parsedImpl, nil
}

// Parses a document, appending it to the list of documents for the repoConfig instance or returning
// an error if the document is invalid.
// @llr REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-56, REQ-TRAQ-SWL-64, REQ-TRAQ-SWL-87
func (rc *RepoConfig) parseDocument(repoName repos.RepoName, doc jsonDoc) error {
	var err error
	parsedDoc := Document{
		Path: doc.Path,
		Schema: Schema{
			Requirements:  nil,
			Attributes:    make(map[string]*Attribute),
			AsmAttributes: make(map[string]*Attribute),
		},
		Implementation: []Implementation{},
	}

	_, err = repos.PathInRepo(repoName, doc.Path)
	if err != nil {
		return errors.Wrapf(err, "Document with path `%s` in repo `%s` cannot be read", doc.Path, repoName)
	}

	parsedDoc.ReqSpec = ReqSpec{Prefix: doc.Prefix, Level: doc.Level}
	parsedDoc.Schema.Requirements, err = regexp.Compile(fmt.Sprintf("(REQ|ASM)-%s-%s-(\\d+)", parsedDoc.ReqSpec.Prefix, parsedDoc.ReqSpec.Level))
	if err != nil {
		return err
	}

	for _, rawAttribute := range doc.Attributes {
		parsedName, parsedAttr, err := parseAttribute(rawAttribute)
		if err != nil {
			return err
		}

		if parsedName == "PARENTS" {
			return fmt.Errorf(`Invalid attribute Parents specified in reqtraq_config.json.
The parents attribute is implicit from the parent declaration in the document`)
		}

		parsedDoc.Schema.Attributes[parsedName] = &parsedAttr
	}

	// Add the parents attribute and link specifications
	if doc.Parent != nil {
		for _, p := range doc.Parent {
			link, err := parseParent(p, doc.Prefix, doc.Level)
			if err != nil {
				return err
			}
			parsedDoc.LinkSpecs = append(parsedDoc.LinkSpecs, link)
		}

	}
	if parsedDoc.LinkSpecs != nil {
		parsedDoc.Schema.Attributes["PARENTS"] = &Attribute{
			Type:  AttributeAny,
			Value: regexp.MustCompile(".*"),
		}
	}

	for _, rawAttribute := range doc.AsmAttributes {
		parsedName, parsedAttr, err := parseAttribute(rawAttribute)
		if err != nil {
			return err
		}

		if parsedName == "PARENTS" {
			return fmt.Errorf(`Invalid attribute Parents specified in reqtraq_config.json for assumptions.
The parents attribute for assumptions is implicit and refers to requirements in the same document`)
		}

		parsedDoc.Schema.AsmAttributes[parsedName] = &parsedAttr
	}

	// Add parents attribute for assumptions
	parsedDoc.Schema.AsmAttributes["PARENTS"] = &Attribute{
		Type:  AttributeRequired,
		Value: regexp.MustCompile(fmt.Sprintf("REQ-%s-%s-(\\d+)", parsedDoc.ReqSpec.Prefix, parsedDoc.ReqSpec.Level)),
	}

	for _, impl := range doc.Implementation {
		parsedImpl, err := parseImplementation(repoName, &impl)
		if err != nil {
			return err
		}
		parsedDoc.Implementation = append(parsedDoc.Implementation, *parsedImpl)
	}

	rc.Documents = append(rc.Documents, parsedDoc)

	return nil
}

// Appends the common attributes to the document and exits with an error if some attribute is
// already defined by the document's attributes.
// @llr REQ-TRAQ-SWL-53
func (doc *Document) appendCommonAttributes(commonAttributes *map[string]*Attribute) error {
	for attrName := range *commonAttributes {
		if _, ok := doc.Schema.Attributes[attrName]; ok {
			return fmt.Errorf("Document with path `%s` redefines attribute with name `%s`, but it is listed as a common attribute",
				doc.Path, attrName)
		}

		doc.Schema.Attributes[attrName] = (*commonAttributes)[attrName]
	}
	return nil
}

// Parses a configuration file into the config instance, recursing into each child (if `DirectDependenciesOnly` is not selected)
// until all configuration files have been parsed. It also parses parent repositories (if any).
// @llr REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-52, REQ-TRAQ-SWL-68
func (config *Config) parseConfigFile(jsonConfig jsonConfig, commonAttributes *map[string]*Attribute) error {
	repoConfig := RepoConfig{}

	// Check if this repo has already been parsed and ignore it
	if _, ok := config.Repos[jsonConfig.RepoName]; ok {
		// Already exists, not parsing it
		return nil
	}

	for _, commonAttr := range jsonConfig.CommonAttributes {
		// Add these to the list in our config
		parsedName, parsedAttr, err := parseAttribute(commonAttr)
		if err != nil {
			return err
		}

		// Double-check that this attribute is not already defined
		if _, ok := (*commonAttributes)[parsedName]; ok {
			return fmt.Errorf("Common attribute with name `%s` found in config for repo `%s` is already defined elsewhere",
				parsedName, jsonConfig.RepoName)
		}

		(*commonAttributes)[parsedName] = &parsedAttr
	}

	for _, doc := range jsonConfig.Docs {
		err := repoConfig.parseDocument(jsonConfig.RepoName, doc)
		if err != nil {
			return err
		}
	}

	config.Repos[jsonConfig.RepoName] = repoConfig

	// Parse any children it has if we are not just checking direct dependencies
	if !DirectDependenciesOnly {
		for _, childRepo := range jsonConfig.ChildrenRepos {
			childRepoPath, err := repos.GetRepo(childRepo.RepoName, childRepo.RemotePath, "", false)
			if err != nil {
				return fmt.Errorf("Error getting child repo name from: %s", childRepo)
			}

			childJsonConfig, err := readJsonConfigFromRepo(childRepoPath)
			if err != nil {
				return err
			}

			// TODO(ja): Validate the name of the child and exit if it does not match
			if childRepo.RepoName != childJsonConfig.RepoName {
				return fmt.Errorf("Configuration for repo `%s` contains child with name `%s` but the url points to a repo with name `%s`",
					jsonConfig.RepoName, childRepo.RepoName, childJsonConfig.RepoName)
			}

			err = config.parseConfigFile(childJsonConfig, commonAttributes)
			if err != nil {
				return err
			}
		}
	}

	// Parse the parent if there is one
	if jsonConfig.ParentRepo.RepoName == "" {
		return nil
	}

	parentRepoPath, err := repos.GetRepo(jsonConfig.ParentRepo.RepoName, jsonConfig.ParentRepo.RemotePath, "", false)
	if err != nil {
		return errors.Wrapf(err, "Error getting repository with path: %s", jsonConfig.ParentRepo)
	}

	parentConfig, err := readJsonConfigFromRepo(parentRepoPath)
	if err != nil {
		return err
	}

	if jsonConfig.ParentRepo.RepoName != parentConfig.RepoName {
		return fmt.Errorf("Repo `%s` defines parent repository with name `%s`, but `%s` was found in url",
			jsonConfig.RepoName, jsonConfig.ParentRepo.RepoName, parentConfig.RepoName)
	}

	return config.parseConfigFile(parentConfig, commonAttributes)
}

// Appends common attributes to each of the document's attributes to build a comprehensive list of
// attributes per document. If any of the documents already contrains the attribute it will exit
// with an error to let the user know about this duplication
// @llr REQ-TRAQ-SWL-53
func (config *Config) appendCommonAttributes(commonAttributes *map[string]*Attribute) error {
	for repoName := range config.Repos {
		for docIndex := range config.Repos[repoName].Documents {
			err := config.Repos[repoName].Documents[docIndex].appendCommonAttributes(commonAttributes)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
