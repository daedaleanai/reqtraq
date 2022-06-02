// Reads configuration data from a reqtraq_config.json file

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/daedaleanai/reqtraq/repos"
)

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

type jsonDoc struct {
	Path           string             `json:"path"`
	Requirements   string             `json:"requirements"`
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

type Document struct {
	Path           string
	Requirements   *regexp.Regexp
	Attributes     map[string]Attribute
	Implementation Implementation
}

type RepoConfig struct {
	Documents []Document
}

type Config struct {
	CommonAttributes map[string]Attribute
	Repos            map[repos.RepoName]RepoConfig
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

	return rawAttribute.Name, attribute, nil
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
		Attributes: make(map[string]Attribute),
		Path:       doc.Path,
	}

	_, err = repos.PathInRepo(repoName, doc.Path)
	if err != nil {
		return fmt.Errorf("Document with path `%s` in repo `%s` cannot be read: %s", doc.Path, repoName, err)
	}

	parsedDoc.Requirements, err = regexp.Compile(doc.Requirements)
	if err != nil {
		return err
	}

	for _, rawAttribute := range doc.Attributes {
		parsedName, parsedAttr, err := parseAttribute(rawAttribute)
		if err != nil {
			return err
		}

		parsedDoc.Attributes[parsedName] = parsedAttr
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

// Parses a configuration file into the config instance, recursing into each children until all
// configuration files have been parsed.
func (config *Config) parseConfigFile(repoName repos.RepoName, jsonConfig jsonConfig) error {

	// Parse this config file
	repoConfig := RepoConfig{}

	for _, commonAttr := range jsonConfig.CommonAttributes {
		// Add these to the list in our config
		parsedName, parsedAttr, err := parseAttribute(commonAttr)
		if err != nil {
			return err
		}

		config.CommonAttributes[parsedName] = parsedAttr
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

		err = config.parseConfigFile(childRepoName, childJsonConfig)
		if err != nil {
			return err
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
		CommonAttributes: make(map[string]Attribute),
		Repos:            make(map[repos.RepoName]RepoConfig),
	}
	err = config.parseConfigFile(topLevelRepoName, topLevelConfig)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
