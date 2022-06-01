// Reads configuration data from a reqtraq_config.json file

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/daedaleanai/reqtraq/repos"
)

type ReqLevel string
type ReqPrefix string

/// Types exported for parsing json files

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
	Code  jsonFileQuery `json:"code"`
	Tests jsonFileQuery `json:"tests"`
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
	Implementation jsonImplementation `json:"implementation"`
}

type jsonConfig struct {
	CommonAttributes []jsonAttribute    `json:"commonAttributes"`
	ParentRepo       repos.RemotePath   `json:"parentRepository"`
	ChildrenRepos    []repos.RemotePath `json:"childrenRepositories"`
	Docs             []jsonDoc          `json:"documents"`
}

/// Types exported for application use

type AttributeType uint

const (
	AttributeRequired AttributeType = iota
	AttributeOptional
	AttributeAny
)

type Attribute struct {
	Type  AttributeType
	Value *regexp.Regexp
}

type Implementation struct {
	CodeFiles []string
	TestFiles []string
}

type Schema struct {
	Requirements *regexp.Regexp
	Attributes   map[string]*Attribute
}

type ReqSpec struct {
	Prefix ReqPrefix
	Level  ReqLevel
}

type Document struct {
	Path           string
	ReqSpec        ReqSpec
	ParentReqSpec  ReqSpec
	Schema         Schema
	Implementation Implementation
}

type RepoConfig struct {
	Documents []Document
}

type Config struct {
	Repos map[repos.RepoName]RepoConfig
}

// Reads a json configuration file from the specify repository path.
// The file is always located at reqtraq_config.json
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

// Finds the top-level configuration file, by searching all config files and its parents until it finds one
// without a parent. It then returns the name of the repository where it was found and its json configuration
func findTopLevelConfig(repoName repos.RepoName, config jsonConfig) (repos.RepoName, jsonConfig, error) {
	if config.ParentRepo != "" {
		parentRepoName, parentRepoPath, err := repos.GetRepo(config.ParentRepo, "", false)
		if err != nil {
			return repoName, jsonConfig{}, fmt.Errorf("Error getting repository with path: %s. %s", config.ParentRepo, err)
		}

		parentConfig, err := readJsonConfigFromRepo(parentRepoPath)
		if err != nil {
			return repoName, jsonConfig{}, err
		}

		return findTopLevelConfig(parentRepoName, parentConfig)
	}

	return repoName, config, nil
}

// Parses an a single attribute from its json description
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
func (rc *RepoConfig) parseDocument(repoName repos.RepoName, doc jsonDoc) error {
	var err error
	parsedDoc := Document{
		Path: doc.Path,
		Schema: Schema{
			Requirements: nil,
			Attributes:   make(map[string]*Attribute),
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
			return fmt.Errorf(`Invalid attribute Parent specified in reqtraq_config.json.
The parent attribute is implicit from the parent declaration in the document`)
		}

		parsedDoc.Schema.Attributes[parsedName] = &parsedAttr
	}

	parsedDoc.ParentReqSpec.Level = doc.Parent.Level
	parsedDoc.ParentReqSpec.Prefix = doc.Parent.Prefix
	if doc.Parent.Level != "" && doc.Parent.Prefix != "" {
		// Add the parent attribute
		parsedDoc.Schema.Attributes["PARENTS"] = &Attribute{
			Type:  AttributeAny,
			Value: regexp.MustCompile(fmt.Sprintf("REQ-%s-%s-(\\d+)", parsedDoc.ParentReqSpec.Prefix, parsedDoc.ParentReqSpec.Level)),
		}
	}

	parsedDoc.Implementation.CodeFiles, err = doc.Implementation.Code.findAllMatchingFiles(repoName)
	if err != nil {
		return err
	}

	parsedDoc.Implementation.TestFiles, err = doc.Implementation.Tests.findAllMatchingFiles(repoName)
	if err != nil {
		return err
	}

	rc.Documents = append(rc.Documents, parsedDoc)

	return nil
}

// Appends the common attributes to the document and exits with an error if some attribute is
// already defined by the document's attributes.
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

// Returns true if the document has a parent
func (doc *Document) HasParent() bool {
	return doc.ParentReqSpec.Level != "" && doc.ParentReqSpec.Prefix != ""
}

// Returns true if the document has associated implementation
func (doc *Document) HasImplementation() bool {
	return len(doc.Implementation.CodeFiles) != 0
}

// Returns true if the document matches the given requirement spec.
func (doc *Document) MatchesSpec(reqSpec ReqSpec) bool {
	return (reqSpec.Level == doc.ReqSpec.Level) && (reqSpec.Prefix == doc.ReqSpec.Prefix)
}

// Converts the requirement specification to a REQ string
func (rs ReqSpec) ToString() string {
	return fmt.Sprintf("REQ-%s-%s", rs.Prefix, rs.Level)
}

// Parses a configuration file into the config instance, recursing into each children until all
// configuration files have been parsed.
func (config *Config) parseConfigFile(repoName repos.RepoName, jsonConfig jsonConfig, commonAttributes *map[string]*Attribute) error {

	// Parse this config file
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
				parsedName, repoName)
		}

		(*commonAttributes)[parsedName] = &parsedAttr
	}

	for _, doc := range jsonConfig.Docs {
		err := repoConfig.parseDocument(repoName, doc)
		if err != nil {
			return err
		}
	}

	config.Repos[repoName] = repoConfig

	// Parse any children it has
	for _, childRepo := range jsonConfig.ChildrenRepos {
		childRepoName, childRepoPath, err := repos.GetRepo(childRepo, "", false)
		if err != nil {
			return fmt.Errorf("Error getting child repo name from: %s", childRepo)
		}

		childJsonConfig, err := readJsonConfigFromRepo(childRepoPath)
		if err != nil {
			return err
		}

		err = config.parseConfigFile(childRepoName, childJsonConfig, commonAttributes)
		if err != nil {
			return err
		}
	}

	return nil
}

// Appends common attributes to each of the document's attributes to build a comprehensive list of
// attributes per document. If any of the documents already contrains the attribute it will exit
// with an error to let the user know about this duplication
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

// Finds the associated Document for a certdoc located at the given path or an error if the
// document is not found
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

/// Builds a map of the linked child -> parent requirement specification to know what specs are related by a
// parent/children relationship
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

// Top level function to parse the configuration file from the given path in the current repository
func ParseConfig(currentRepoPath string) (Config, error) {
	repoName := repos.GetRepoNameFromPath(currentRepoPath)
	repoPath, err := repos.GetRepoPathByName(repoName)
	if err != nil {
		return Config{}, err
	}

	jsonConfig, err := readJsonConfigFromRepo(repoPath)
	if err != nil {
		return Config{}, err
	}

	// If this is not the top level configuration we need to clone the parent repo and handle requirements from there
	// Find the top level config, then start parsing them
	topLevelRepoName, topLevelConfig, err := findTopLevelConfig(repoName, jsonConfig)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		Repos: make(map[repos.RepoName]RepoConfig),
	}

	commonAttributes := make(map[string]*Attribute)

	err = config.parseConfigFile(topLevelRepoName, topLevelConfig, &commonAttributes)
	if err != nil {
		return Config{}, err
	}

	config.appendCommonAttributes(&commonAttributes)

	return config, nil
}
