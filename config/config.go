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

type jsonFileQuery struct {
	Paths           []string `json:"paths"`
	MatchingPattern string   `json:"matchingPattern"`
	IgnoredPatterns []string `json:"ignoredPatterns"`
}

type jsonImplementation struct {
	Code                jsonFileQuery `json:"code"`
	Tests               jsonFileQuery `json:"tests"`
	CompilationDatabase string        `json:"compilationDatabase"`
	ClangArguments      []string      `json:"clangArguments"`
}

type jsonParent struct {
	Prefix ReqPrefix `json:"prefix"`
	Level  ReqLevel  `json:"level"`
}

type jsonDoc struct {
	Path           string             `json:"path"`
	Prefix         ReqPrefix          `json:"prefix"`
	Level          ReqLevel           `json:"level"`
	Parent         jsonParent         `json:"parent"`
	Attributes     []jsonAttribute    `json:"attributes"`
	AsmAttributes  []jsonAttribute    `json:"asmAttributes"`
	Implementation jsonImplementation `json:"implementation"`
}

type jsonConfig struct {
	RepoName         repos.RepoName  `json:"repoName"`
	CommonAttributes []jsonAttribute `json:"commonAttributes"`
	ParentRepo       jsonRepoLink    `json:"parentRepository"`
	ChildrenRepos    []jsonRepoLink  `json:"childrenRepositories"`
	Docs             []jsonDoc       `json:"documents"`
	PreferLibClang   bool            `json:"preferLibClang"`
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

// A structure describing the implementation for a given certification document.
type Implementation struct {
	CodeFiles           []string
	TestFiles           []string
	CompilationDatabase string
	ClangArguments      []string
}

// The schema for requirements inside a certification document
type Schema struct {
	Requirements  *regexp.Regexp
	Attributes    map[string]*Attribute
	AsmAttributes map[string]*Attribute
}

// A requirement specification. Identifies the form of requirements in a document
type ReqSpec struct {
	Prefix ReqPrefix
	Level  ReqLevel
}

// A certification document with its given requirement specification and schema, as well as its
// implementation in terms of code and its location in the repository
type Document struct {
	Path           string
	ReqSpec        ReqSpec
	ParentReqSpec  ReqSpec
	Schema         Schema
	Implementation Implementation
}

// A configuration for a single repository, which is made of documents.
type RepoConfig struct {
	Documents []Document
}

// A global configuration structure for all repositories that compose the system.
type Config struct {
	Repos          map[repos.RepoName]RepoConfig
	PreferLibClang bool
}

// Top level function to parse the configuration file from the given path in the current repository
// @llr REQ-TRAQ-SWL-53
func ParseConfig(repoPath repos.RepoPath) (Config, error) {
	jsonConfig, err := readJsonConfigFromRepo(repoPath)
	if err != nil {
		return Config{}, errors.Wrapf(err, "The requested config path `%s` does not contain a valid repository", repoPath)
	}

	// If this is not the top level configuration we need to clone the parent repo and handle requirements from there
	// Find the top level config, then start parsing them
	topLevelConfig, err := findTopLevelConfig(jsonConfig)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Repos: make(map[repos.RepoName]RepoConfig),
	}

	commonAttributes := make(map[string]*Attribute)

	err = config.parseConfigFile(topLevelConfig, &commonAttributes)
	if err != nil {
		return Config{}, err
	}

	config.appendCommonAttributes(&commonAttributes)

	return config, nil
}

// Returns true if the document has a parent
// @llr REQ-TRAQ-SWL-55
func (doc *Document) HasParent() bool {
	return doc.ParentReqSpec.Level != "" && doc.ParentReqSpec.Prefix != ""
}

// Returns true if the document has associated implementation
// @llr REQ-TRAQ-SWL-56
func (doc *Document) HasImplementation() bool {
	return len(doc.Implementation.CodeFiles) != 0
}

// Returns true if the document matches the given requirement spec.
// @llr REQ-TRAQ-SWL-55
func (doc *Document) MatchesSpec(reqSpec ReqSpec) bool {
	return (reqSpec.Level == doc.ReqSpec.Level) && (reqSpec.Prefix == doc.ReqSpec.Prefix)
}

// Converts the requirement specification to a REQ string
// @llr REQ-TRAQ-SWL-55
func (rs ReqSpec) ToString() string {
	return fmt.Sprintf("REQ-%s-%s", rs.Prefix, rs.Level)
}

// Finds the associated Document for a certdoc located at the given path or a nil document if it
// is not found
// @llr REQ-TRAQ-SWL-54
func (config *Config) FindCertdoc(path string) (repos.RepoName, *Document) {
	for repoName := range config.Repos {
		for docIdx := range config.Repos[repoName].Documents {
			if config.Repos[repoName].Documents[docIdx].Path == path {
				return repoName, &config.Repos[repoName].Documents[docIdx]
			}
		}
	}
	return "", nil
}

// Builds a map of the linked child -> parent requirement specification to know what specs are related by a
// parent/children relationship
// @llr REQ-TRAQ-SWL-54
func (config *Config) GetLinkedReqSpecs() map[ReqSpec]ReqSpec {
	links := make(map[ReqSpec]ReqSpec)

	for repoName := range config.Repos {
		for docIdx := range config.Repos[repoName].Documents {
			doc := &config.Repos[repoName].Documents[docIdx]
			if doc.HasParent() {
				links[doc.ReqSpec] = doc.ParentReqSpec
			}
		}
	}

	return links
}

// Loads the information for the base repository from git
// @llr REQ-TRAQ-SWL-53
func LoadBaseRepoInfo() {
	// See details about "working directory" in https://git-scm.com/docs/githooks
	bare, err := linepipes.Single(linepipes.Run("git", "rev-parse", "--is-bare-repository"))
	if err != nil {
		log.Fatalf("Failed to check Git repository type. Are you running reqtraq in a Git repo?\n%s", err)
	}
	if bare == "true" {
		log.Fatal("Reqtraq cannot be used in bare checkouts")
	}

	toplevel, err := linepipes.Single(linepipes.Run("git", "rev-parse", "--show-toplevel"))
	if err != nil {
		log.Fatal(err)
	}

	basePath := repos.RepoPath(toplevel)

	config, err := readJsonConfigFromRepo(basePath)
	if err != nil {
		log.Fatalf("Error reading configuration in path: %s, %v", basePath, err)
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
		return jsonConfig{}, fmt.Errorf("Error while parsing configuration file `%s`: %s", configPath, err)
	}
	return config, nil
}

// Finds the top-level configuration file, by searching all config files and their parents until it finds one
// without a parent.
// @llr REQ-TRAQ-SWL-52
func findTopLevelConfig(config jsonConfig) (jsonConfig, error) {
	if config.ParentRepo.RepoName != "" {
		parentRepoPath, err := repos.GetRepo(config.ParentRepo.RepoName, config.ParentRepo.RemotePath, "", false)
		if err != nil {
			return jsonConfig{}, fmt.Errorf("Error getting repository with path: %s. %s", config.ParentRepo, err)
		}

		parentConfig, err := readJsonConfigFromRepo(parentRepoPath)
		if err != nil {
			return jsonConfig{}, err
		}

		if config.ParentRepo.RepoName != parentConfig.RepoName {
			return jsonConfig{}, fmt.Errorf("Repo `%s` defines parent repository with name `%s`, but `%s` was found in url",
				config.RepoName, config.ParentRepo.RepoName, parentConfig.RepoName)
		}

		return findTopLevelConfig(parentConfig)
	}

	return config, nil
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

// Finds all matching files for the given query under the given repository.
// @llr REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-56
func (fileQuery *jsonFileQuery) findAllMatchingFiles(repoName repos.RepoName) ([]string, error) {
	var matchingPattern *regexp.Regexp = nil
	if fileQuery.MatchingPattern != "" {
		var err error
		matchingPattern, err = regexp.Compile(fileQuery.MatchingPattern)
		if err != nil {
			return []string{}, err
		}
	}

	var collectedFiles = []string{}

	var ignoredPatterns []*regexp.Regexp
	for _, pattern := range fileQuery.IgnoredPatterns {
		compiledPattern, err := regexp.Compile(pattern)
		if err != nil {
			return []string{}, fmt.Errorf("Unable to parse `%s` as a regular expression", pattern)
		}
		ignoredPatterns = append(ignoredPatterns, compiledPattern)
	}

	for _, path := range fileQuery.Paths {
		matched_files, err := repos.FindFilesInDirectory(repoName, path, matchingPattern, ignoredPatterns)
		if err != nil {
			return []string{}, err
		}
		collectedFiles = append(collectedFiles, matched_files...)
	}

	return collectedFiles, nil
}

// Parses a document, appending it to the list of documents for the repoConfig instance or returning
// an error if the document is invalid.
// @llr REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-56, REQ-TRAQ-SWL-64
func (rc *RepoConfig) parseDocument(repoName repos.RepoName, doc jsonDoc) error {
	var err error
	parsedDoc := Document{
		Path: doc.Path,
		Schema: Schema{
			Requirements:  nil,
			Attributes:    make(map[string]*Attribute),
			AsmAttributes: make(map[string]*Attribute),
		},
	}

	_, err = repos.PathInRepo(repoName, doc.Path)
	if err != nil {
		return fmt.Errorf("Document with path `%s` in repo `%s` cannot be read: %s", doc.Path, repoName, err)
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

	parsedDoc.ParentReqSpec.Level = doc.Parent.Level
	parsedDoc.ParentReqSpec.Prefix = doc.Parent.Prefix
	if doc.Parent.Level != "" && doc.Parent.Prefix != "" {
		// Add the parents attribute
		parsedDoc.Schema.Attributes["PARENTS"] = &Attribute{
			Type:  AttributeAny,
			Value: regexp.MustCompile(fmt.Sprintf("REQ-%s-%s-(\\d+)", parsedDoc.ParentReqSpec.Prefix, parsedDoc.ParentReqSpec.Level)),
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

	parsedDoc.Implementation.CodeFiles, err = doc.Implementation.Code.findAllMatchingFiles(repoName)
	if err != nil {
		return err
	}

	parsedDoc.Implementation.TestFiles, err = doc.Implementation.Tests.findAllMatchingFiles(repoName)
	if err != nil {
		return err
	}
	parsedDoc.Implementation.CompilationDatabase = doc.Implementation.CompilationDatabase
	parsedDoc.Implementation.ClangArguments = doc.Implementation.ClangArguments
	if parsedDoc.Implementation.ClangArguments == nil {
		parsedDoc.Implementation.ClangArguments = []string{}
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

// Parses a configuration file into the config instance, recursing into each child until all
// configuration files have been parsed.
// @llr REQ-TRAQ-SWL-53
func (config *Config) parseConfigFile(jsonConfig jsonConfig, commonAttributes *map[string]*Attribute) error {
	repoConfig := RepoConfig{}

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

	// Parse any children it has
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

	if jsonConfig.PreferLibClang {
		config.PreferLibClang = true
	}

	return nil
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
