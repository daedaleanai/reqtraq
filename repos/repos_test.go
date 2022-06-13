package repos

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Other packages (config) are expected to do this, but for the repos config we can do it here
// @llr REQ-TRAQ-SWL-49
func TestMain(m *testing.M) {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get current directory")
	}

	SetBaseRepoInfo(RepoPath(filepath.Dir(workingDir)), RepoName("reqtraq"))

}

// @llr REQ-TRAQ-SWL-49
func TestRepos_BaseRepoName(t *testing.T) {
	assert.Equal(t, BaseRepoName(), RepoName("reqtraq"))
}

// @llr REQ-TRAQ-SWL-49
func TestRepos_BaseRepoPath(t *testing.T) {
	workingDir, err := os.Getwd()
	assert.Equal(t, err, nil)
	assert.Equal(t, BaseRepoPath(), filepath.Dir(workingDir))
}

// @llr REQ-TRAQ-SWL-49
func TestRepos_RegisterRepository(t *testing.T) {
	ClearAllRepositories()

	repoName := RepoName("MyCoolRepo")
	RegisterRepository(repoName, RepoPath("/some/fake/path/MyCoolRepo"))

	path, err := GetRepoPathByName(repoName)
	assert.Equal(t, err, nil)
	assert.Equal(t, path, RepoPath("/some/fake/path/MyCoolRepo"))
}

// @llr REQ-TRAQ-SWL-49
func TestRepos_ClearAllRepositories(t *testing.T) {
	ClearAllRepositories()

	repoName := RepoName("MyCoolRepo")
	RegisterRepository(repoName, "/some/fake/path/MyCoolRepo")
	assert.Equal(t, repoName, RepoName("MyCoolRepo"))

	path, err := GetRepoPathByName(repoName)
	assert.Equal(t, err, nil)
	assert.Equal(t, path, RepoPath("/some/fake/path/MyCoolRepo"))

	ClearAllRepositories()
	path, err = GetRepoPathByName(repoName)
	assert.NotEqual(t, err, nil)
}

// @llr REQ-TRAQ-SWL-49
func TestRepos_GetRepo_NoOverrideRegistered(t *testing.T) {
	baseRepoPath := BaseRepoPath()
	baseRepoName := BaseRepoName()
	ClearAllRepositories()
	RegisterRepository(baseRepoName, baseRepoPath)

	path, err := GetRepo(baseRepoName, RemotePath("https://github.com/daedaleanai/reqtraq.git"), "", false)
	assert.Equal(t, err, nil)
	assert.Equal(t, path, RepoPath(baseRepoPath))
}

// @llr REQ-TRAQ-SWL-49
func TestRepos_GetRepo_NoOverrideNotRegistered(t *testing.T) {
	baseRepoName := BaseRepoName()
	ClearAllRepositories()

	tempDirPrefix := filepath.Join(os.TempDir(), ".reqtraq")

	path, err := GetRepo(baseRepoName, RemotePath(BaseRepoPath()), "", false)
	assert.Equal(t, err, nil)

	assert.True(t, strings.HasPrefix(string(path), tempDirPrefix))
}

// @llr REQ-TRAQ-SWL-50
func TestRepos_GetRepo_OverrideRegistered(t *testing.T) {
	baseRepoPath := BaseRepoPath()
	baseRepoName := BaseRepoName()
	ClearAllRepositories()
	RegisterRepository(baseRepoName, baseRepoPath)

	tempDirPrefix := filepath.Join(os.TempDir(), ".reqtraq")

	path, err := GetRepo(baseRepoName, RemotePath(BaseRepoPath()), "", true)
	assert.Equal(t, err, nil)

	assert.True(t, strings.HasPrefix(string(path), tempDirPrefix))
}

// @llr REQ-TRAQ-SWL-49, REQ-TRAQ-SWL-51
func TestRepos_FindFilesInDirectory(t *testing.T) {
	baseRepoPath := BaseRepoPath()
	baseRepoName := BaseRepoName()
	ClearAllRepositories()
	RegisterRepository(baseRepoName, baseRepoPath)

	files, err := FindFilesInDirectory(baseRepoName, "testdata/projectB", regexp.MustCompile(".*"), []*regexp.Regexp{})
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

	files, err = FindFilesInDirectory(baseRepoName, "testdata/projectB", regexp.MustCompile(".*\\.(cc|hh)"), []*regexp.Regexp{})
	assert.Equal(t, err, nil)
	assert.ElementsMatch(t, files, []string{
		"testdata/projectB/code/include/a.hh",
		"testdata/projectB/code/a.cc",
		"testdata/projectB/code/file_ignored.cc",
		"testdata/projectB/test/not_a_test_file.cc",
		"testdata/projectB/test/a/a_test.cc",
	})

	files, err = FindFilesInDirectory(baseRepoName, "testdata/projectB", regexp.MustCompile(".*\\.(cc|hh)"), []*regexp.Regexp{regexp.MustCompile(".*_test\\.(cc|hh)$")})
	assert.Equal(t, err, nil)
	assert.ElementsMatch(t, files, []string{
		"testdata/projectB/code/include/a.hh",
		"testdata/projectB/code/a.cc",
		"testdata/projectB/code/file_ignored.cc",
		"testdata/projectB/test/not_a_test_file.cc",
	})
}

// @llr REQ-TRAQ-SWL-49, REQ-TRAQ-SWL-51
func TestRepos_PathInRepo(t *testing.T) {
	baseRepoPath := BaseRepoPath()
	baseRepoName := BaseRepoName()
	ClearAllRepositories()
	RegisterRepository(baseRepoName, baseRepoPath)

	path, err := PathInRepo(baseRepoName, "testdata/projectB/code/a.cc")
	assert.Equal(t, err, nil)
	assert.Equal(t, path, filepath.Join(string(baseRepoPath), "testdata/projectB/code/a.cc"))

	// Now try a file that does not exist
	path, err = PathInRepo(baseRepoName, "testdata/projectB/code/b.cc")
	assert.NotEqual(t, err, nil)
}

// @llr REQ-TRAQ-SWL-16
func TestRepos_AllCommits(t *testing.T) {
	baseRepoPath := BaseRepoPath()
	baseRepoName := BaseRepoName()
	ClearAllRepositories()
	RegisterRepository(baseRepoName, baseRepoPath)

	commits, err := AllCommits(baseRepoName)
	assert.Equal(t, err, nil)
	assert.NotEmpty(t, commits)

	commitLineMatcher := regexp.MustCompile(`[0-9a-f]{7}\s[0-9]{4}-[0-9]{2}-[0-9]{2}`)

	for _, commit := range commits {
		assert.True(t, commitLineMatcher.MatchString(commit))
	}
}
