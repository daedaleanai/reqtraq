package repos

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepos_GetRepoNameFromPath(t *testing.T) {
	assert.Equal(t, GetRepoNameFromPath("git@github.com:daedaleanai/reqtraq.git"), RepoName("reqtraq"))
	assert.Equal(t, GetRepoNameFromPath("https://github.com/daedaleanai/reqtraq.git"), RepoName("reqtraq"))
	assert.Equal(t, GetRepoNameFromPath("/some/folder/in/my/filesystem/reqtraq"), RepoName("reqtraq"))
}

func TestRepos_BaseRepoName(t *testing.T) {
	assert.Equal(t, BaseRepoName(), RepoName("reqtraq"))
}

func TestRepos_BaseRepoPath(t *testing.T) {
	workingDir, err := os.Getwd()
	assert.Equal(t, err, nil)
	assert.Equal(t, BaseRepoPath(), filepath.Dir(workingDir))
}

func TestRepos_RegisterRepository(t *testing.T) {
	ClearAllRepositories()

	repoName := RegisterRepository("/some/fake/path/MyCoolRepo")
	assert.Equal(t, repoName, RepoName("MyCoolRepo"))

	path, err := GetRepoPathByName(repoName)
	assert.Equal(t, err, nil)
	assert.Equal(t, path, RepoPath("/some/fake/path/MyCoolRepo"))
}

func TestRepos_ClearAllRepositories(t *testing.T) {
	ClearAllRepositories()

	repoName := RegisterRepository("/some/fake/path/MyCoolRepo")
	assert.Equal(t, repoName, RepoName("MyCoolRepo"))

	path, err := GetRepoPathByName(repoName)
	assert.Equal(t, err, nil)
	assert.Equal(t, path, RepoPath("/some/fake/path/MyCoolRepo"))

	ClearAllRepositories()
	path, err = GetRepoPathByName(repoName)
	assert.NotEqual(t, err, nil)
}

func TestRepos_GetRepo_NoOverrideRegistered(t *testing.T) {
	baseRepoPath := BaseRepoPath()
	ClearAllRepositories()
	RegisterRepository(baseRepoPath)

	name, path, err := GetRepo("https://github.com/daedaleanai/reqtraq.git", "", false)
	assert.Equal(t, err, nil)
	assert.Equal(t, name, RepoName("reqtraq"))
	assert.Equal(t, path, RepoPath(baseRepoPath))
}

func TestRepos_GetRepo_NoOverrideNotRegistered(t *testing.T) {
	ClearAllRepositories()

	tempDirPrefix := filepath.Join(os.TempDir(), ".reqtraq")

	name, path, err := GetRepo(RemotePath(BaseRepoPath()), "", false)
	assert.Equal(t, err, nil)
	assert.Equal(t, name, RepoName("reqtraq"))

	assert.True(t, strings.HasPrefix(string(path), tempDirPrefix))
}

func TestRepos_GetRepo_OverrideRegistered(t *testing.T) {
	ClearAllRepositories()
	RegisterRepository(BaseRepoPath())

	tempDirPrefix := filepath.Join(os.TempDir(), ".reqtraq")

	name, path, err := GetRepo(RemotePath(BaseRepoPath()), "", true)
	assert.Equal(t, err, nil)
	assert.Equal(t, name, RepoName("reqtraq"))

	assert.True(t, strings.HasPrefix(string(path), tempDirPrefix))
}

func TestRepos_FindFilesInDirectory(t *testing.T) {
	ClearAllRepositories()
	repoName := RegisterRepository(BaseRepoPath())

	files, err := FindFilesInDirectory(repoName, "testdata/projectB", regexp.MustCompile(".*"), []*regexp.Regexp{})
	assert.Equal(t, err, nil)
	assert.ElementsMatch(t, files, []string{
		"testdata/projectB/TEST-138-SDD.md",
		"testdata/projectB/reqtraq_config.json",
		"testdata/projectB/code/include/a.hh",
		"testdata/projectB/code/a.cc",
		"testdata/projectB/code/file_ignored.cc",
		"testdata/projectB/test/not_a_test_file.cc",
		"testdata/projectB/test/a/a_test.cc",
	})

	files, err = FindFilesInDirectory(repoName, "testdata/projectB", regexp.MustCompile(".*\\.(cc|hh)"), []*regexp.Regexp{})
	assert.Equal(t, err, nil)
	assert.ElementsMatch(t, files, []string{
		"testdata/projectB/code/include/a.hh",
		"testdata/projectB/code/a.cc",
		"testdata/projectB/code/file_ignored.cc",
		"testdata/projectB/test/not_a_test_file.cc",
		"testdata/projectB/test/a/a_test.cc",
	})

	files, err = FindFilesInDirectory(repoName, "testdata/projectB", regexp.MustCompile(".*\\.(cc|hh)"), []*regexp.Regexp{regexp.MustCompile(".*_test\\.(cc|hh)$")})
	assert.Equal(t, err, nil)
	assert.ElementsMatch(t, files, []string{
		"testdata/projectB/code/include/a.hh",
		"testdata/projectB/code/a.cc",
		"testdata/projectB/code/file_ignored.cc",
		"testdata/projectB/test/not_a_test_file.cc",
	})
}

func TestRepos_PathInRepo(t *testing.T) {
	ClearAllRepositories()
	repoName := RegisterRepository(BaseRepoPath())

	path, err := PathInRepo(repoName, "testdata/projectB/code/a.cc")
	assert.Equal(t, err, nil)
	assert.Equal(t, path, filepath.Join(BaseRepoPath(), "testdata/projectB/code/a.cc"))

	// Now try a file that does not exist
	path, err = PathInRepo(repoName, "testdata/projectB/code/b.cc")
	assert.NotEqual(t, err, nil)
}

func TestRepos_AllCommits(t *testing.T) {
	ClearAllRepositories()
	repoName := RegisterRepository(BaseRepoPath())

	commits, err := AllCommits(repoName)
	assert.Equal(t, err, nil)
	assert.NotEmpty(t, commits)

	commitLineMatcher := regexp.MustCompile(`[0-9a-f]{7}\s[0-9]{4}-[0-9]{2}-[0-9]{2}`)

	for id, commit := range commits {
		if id == 0 {
			fmt.Println(commit)
		}
		assert.True(t, commitLineMatcher.MatchString(commit))
	}
}
