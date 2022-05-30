// Reads configuration data from a reqtraq_config.json file

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/daedaleanai/reqtraq/repos"
)

/// Types exported for parsing json files

type jsonAttribute struct {
	Name      string `json:"name"`
	Required  string `json:"required"`
	Value     string `json:"value"`
}

type jsonFileQuery struct {
	Paths              []string  `json:"paths"`
	MatchingPattern    string    `json:"matchingPattern"`
	IgnoredPatterns    []string  `json:"ignoredPatterns"`
}

type jsonImplementation struct {
	Code       jsonFileQuery  `json:"code"`
	Tests      jsonFileQuery  `json:"tests"`
}

type jsonDoc struct {
	Path           string             `json:"path"`
	Requirements   string             `json:"requirements"`
	Attributes     []jsonAttribute    `json:"attributes"`
	Implementation jsonImplementation `json:"implementation"`
}

type jsonConfig struct {
	CommonAttributes []jsonAttribute     `json:"commonAttributes"`
	ParentRepo       repos.RemotePath    `json:"parentRepository"`
	ChildrenRepos    []repos.RemotePath  `json:"childrenRepositories"`
	Docs             []jsonDoc           `json:"documents"`
}

/// Types exported for application use

type AttributeType uint
const (
	AttributeRequired AttributeType = iota
	AttributeOptional
	AttributeAny
)

type Attribute struct {
	Type      AttributeType
	Value     *regexp.Regexp
}

type Implementation struct {
	CodeFiles []string
	TestFiles []string
}

type Document struct {
	Path              string
	Requirements      *regexp.Regexp
	Attributes        map[string]Attribute
	Implementation    Implementation
}

type RepoConfig struct {
	Documents  []Document
}

type Config struct {
	CommonAttributes map[string]Attribute
	Repos map[repos.RepoName]RepoConfig
}

func readJsonConfigFromRepo(repoPath repos.RepoPath) (jsonConfig, error) {
	// Read parent config and parse that
	configPath := fmt.Sprintf("%s/reqtraq_config.json", repoPath)

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

func findTopLevelConfig(repoName repos.RepoName, config jsonConfig) (repos.RepoName, jsonConfig, error) {
	if config.ParentRepo != "" {
		parentRepoName, parentRepoPath, err := repos.GetRepoPathByRemotePath(config.ParentRepo)
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

func (rc *RepoConfig) parseDocument(repoName repos.RepoName, doc jsonDoc) (error) {
	var err error
	parsedDoc := Document{
		Attributes: make(map[string]Attribute),
		Path: doc.Path,
	}

	err = repos.ValidatePath(repoName, doc.Path)
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
			return  err;
		}

		parsedDoc.Attributes[parsedName] = parsedAttr
	}

	parsedDoc.Implementation.CodeFiles, err = doc.Implementation.Code.findAllMatchingFiles(repoName)
	if err != nil {
		return err;
	}

    parsedDoc.Implementation.TestFiles, err = doc.Implementation.Tests.findAllMatchingFiles(repoName)
	if err != nil {
		return err;
	}

	rc.Documents = append(rc.Documents, parsedDoc)

	return nil
}

func (config *Config) parseConfigFile(repoName repos.RepoName, jsonConfig jsonConfig) error {

	// Parse this config file
	repoConfig := RepoConfig{};

	for _, commonAttr := range jsonConfig.CommonAttributes {
		// Add these to the list in our config
		parsedName, parsedAttr, err := parseAttribute(commonAttr)
		if err != nil {
			return  err;
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
		childRepoName, childRepoPath, err := repos.GetRepoPathByRemotePath(childRepo)
		if err != nil {
			return fmt.Errorf("Error getting child repo name from: %s", childRepo)
		}

		childJsonConfig, err := readJsonConfigFromRepo(childRepoPath)
		if err != nil {
			return err
		}

		err = config.parseConfigFile(childRepoName, childJsonConfig);
		if err != nil {
			return err
		}
	}

	return nil
}

// Top level function to parse the configuration file from the given path in the current repository
func ParseConfig(currentRepoPath string) (Config, error) {
	jsonConfig, err := readJsonConfigFromRepo(repos.RepoPath(currentRepoPath))
	if err != nil {
		return Config{}, err
	}

	// Register ourselves (Or the current repo path) in the registry of repositories
	repoName := repos.RegisterCurrentRepository(currentRepoPath)

	// If this is not the top level configuration we need to clone the parent repo and handle requirements from there
	// Find the top level config, then start parsing them
	topLevelRepoName, topLevelConfig, err := findTopLevelConfig(repoName, jsonConfig)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		CommonAttributes: make(map [string]Attribute),
		Repos: make(map [repos.RepoName]RepoConfig),
	}
	err = config.parseConfigFile(topLevelRepoName, topLevelConfig)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}