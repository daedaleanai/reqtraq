package report

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/daedaleanai/reqtraq/reqs"
	"github.com/stretchr/testify/assert"
)

// Other packages (config) are expected to do this, but for the repos config we can do it here
// @llr REQ-TRAQ-SWL-49
func TestMain(m *testing.M) {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get current directory")
	}

	repos.SetBaseRepoInfo(repos.RepoPath(filepath.Dir(workingDir)), repos.RepoName("reqtraq"))
	os.Exit(m.Run())
}

// @llr REQ-TRAQ-SWL-12, REQ-TRAQ-SWL-13, REQ-TRAQ-SWL-20, REQ-TRAQ-SWL-21, REQ-TRAQ-SWL-30, REQ-TRAQ-SWL-31
func TestReports(t *testing.T) {
	repos.ClearAllRepositories()
	repos.RegisterRepository(repos.BaseRepoName(), repos.BaseRepoPath())
	reqtraqConfig, err := config.ParseConfig(repos.BaseRepoPath())
	if err != nil {
		t.Fatal(err)
	}

	rg, err := reqs.BuildGraph("", &reqtraqConfig)
	if err != nil {
		t.Fatal(err)
	}

	{
		if err := ReportDown(rg, ioutil.Discard); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := ReportUp(rg, ioutil.Discard); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := ReportIssues(rg, ioutil.Discard); err != nil {
			t.Fatal(err)
		}
	}

	{
		var filter reqs.ReqFilter
		filter.IDRegexp = regexp.MustCompile("10")
		checkFilteredReports(t, rg, &filter)
	}
	{
		var filter reqs.ReqFilter
		filter.TitleRegexp = regexp.MustCompile("navigation")
		checkFilteredReports(t, rg, &filter)
	}
	{
		var filter reqs.ReqFilter
		filter.BodyRegexp = regexp.MustCompile("heading")
		checkFilteredReports(t, rg, &filter)
	}
	{
		var filter reqs.ReqFilter
		filter.AnyAttributeRegexp = regexp.MustCompile("navigation")
		checkFilteredReports(t, rg, &filter)
	}
	{
		var filter reqs.ReqFilter
		filter.AttributeRegexp = make(map[string]*regexp.Regexp)
		filter.AttributeRegexp["VERIFICATION"] = regexp.MustCompile("Demo")
		checkFilteredReports(t, rg, &filter)
	}
}

// @llr REQ-TRAQ-SWL-20, REQ-TRAQ-SWL-21, REQ-TRAQ-SWL-31
func checkFilteredReports(t *testing.T, rg *reqs.ReqGraph, filter *reqs.ReqFilter) {
	var diffs map[string][]string

	{
		if err := ReportDownFiltered(rg, ioutil.Discard, filter, diffs); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := ReportUpFiltered(rg, ioutil.Discard, filter, diffs); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := ReportIssuesFiltered(rg, ioutil.Discard, filter, diffs); err != nil {
			t.Fatal(err)
		}
	}
}

// @llr REQ-TRAQ-SWL-20, REQ-TRAQ-SWL-21, REQ-TRAQ-SWL-31, REQ-TRAQ-SWL-19
func TestReport_Matches_filter(t *testing.T) {
	tests := []struct {
		req     reqs.Req
		filter  reqs.ReqFilter
		diffs   map[string][]string
		matches bool
	}{
		{reqs.Req{ID: "REQ-TEST-SWH-1", Body: "thrust control"},
			reqs.ReqFilter{IDRegexp: regexp.MustCompile("REQ-TEST-SWH-*")},
			nil,
			true},
		{reqs.Req{ID: "REQ-TEST-SWH-1", Title: "The control unit will calculate thrust.", Body: "It will also do much more."},
			reqs.ReqFilter{TitleRegexp: regexp.MustCompile("thrust")},
			nil,
			true},
		{reqs.Req{ID: "REQ-TEST-SWH-1", Title: "The control unit will calculate vertical take off speed.", Body: "It will also output thrust."},
			reqs.ReqFilter{TitleRegexp: regexp.MustCompile("thrust")},
			nil,
			false},
		{reqs.Req{ID: "REQ-TEST-SWH-1", Body: "thrust control"},
			reqs.ReqFilter{BodyRegexp: regexp.MustCompile("thrust")},
			nil,
			true},
		{reqs.Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			reqs.ReqFilter{IDRegexp: regexp.MustCompile("REQ-*"), BodyRegexp: regexp.MustCompile("thrust")},
			nil,
			true},
		{reqs.Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			reqs.ReqFilter{IDRegexp: regexp.MustCompile("REQ-DDLN-*"), BodyRegexp: regexp.MustCompile("thrust")},
			nil,
			false},

		// filter attributes
		{reqs.Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			reqs.ReqFilter{AnyAttributeRegexp: regexp.MustCompile("Demo*")},
			nil,
			true},
		{reqs.Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			reqs.ReqFilter{AnyAttributeRegexp: regexp.MustCompile("Test*")},
			nil,
			false},
		{reqs.Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			reqs.ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Demo*")}},
			nil,
			true},
		{reqs.Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Color": "Brown"}},
			reqs.ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Demo*")}},
			nil,
			false},
		{reqs.Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			reqs.ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Test*")}},
			nil,
			false},

		// diffs
		{reqs.Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			reqs.ReqFilter{},
			map[string][]string{"REQ-TEST-SWL-1": make([]string, 0)},
			false},
		{reqs.Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			reqs.ReqFilter{},
			map[string][]string{"REQ-TEST-SWL-14": make([]string, 0)},
			true},
		{reqs.Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			reqs.ReqFilter{IDRegexp: regexp.MustCompile("X")},
			map[string][]string{"REQ-TEST-SWL-14": make([]string, 0)},
			false},
	}

	for _, test := range tests {
		if test.matches {
			assert.True(t, test.req.Matches(&test.filter, test.diffs), "expected requirement to match: %v filter=%v diffs=%v", test.req, test.filter, test.diffs)
		} else {
			assert.False(t, test.req.Matches(&test.filter, test.diffs), "expected requirement to not match: %v filter=%v diffs=%v", test.req, test.filter, test.diffs)
		}
	}
}
