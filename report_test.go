package main

import (
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReports(t *testing.T) {
	rg, err := CreateReqGraph(*fCertdocPath, *fCodePath, *fSchemaPath)
	if err != nil {
		t.Fatal(err)
	}

	{
		if err := rg.ReportDown(ioutil.Discard); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := rg.ReportUp(ioutil.Discard); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := rg.ReportIssues(ioutil.Discard); err != nil {
			t.Fatal(err)
		}
	}

	{
		var filter ReqFilter
		filter.IDRegexp = regexp.MustCompile("10")
		checkFilteredReports(t, rg, &filter)
	}
	{
		var filter ReqFilter
		filter.TitleRegexp = regexp.MustCompile("navigation")
		checkFilteredReports(t, rg, &filter)
	}
	{
		var filter ReqFilter
		filter.BodyRegexp = regexp.MustCompile("heading")
		checkFilteredReports(t, rg, &filter)
	}
	{
		var filter ReqFilter
		filter.AnyAttributeRegexp = regexp.MustCompile("navigation")
		checkFilteredReports(t, rg, &filter)
	}
	{
		var filter ReqFilter
		filter.AttributeRegexp = make(map[string]*regexp.Regexp)
		filter.AttributeRegexp["VERIFICATION"] = regexp.MustCompile("Demo")
		checkFilteredReports(t, rg, &filter)
	}
}

func checkFilteredReports(t *testing.T, rg *ReqGraph, filter *ReqFilter) {
	var diffs map[string][]string

	{
		if err := rg.ReportDownFiltered(ioutil.Discard, filter, diffs); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := rg.ReportUpFiltered(ioutil.Discard, filter, diffs); err != nil {
			t.Fatal(err)
		}
	}
	{
		if err := rg.ReportIssuesFiltered(ioutil.Discard, filter, diffs); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReport_Matches_filter(t *testing.T) {
	tests := []struct {
		req     Req
		filter  ReqFilter
		diffs   map[string][]string
		matches bool
	}{
		{Req{ID: "REQ-TEST-SWH-1", Body: "thrust control"},
			ReqFilter{IDRegexp: regexp.MustCompile("REQ-TEST-SWH-*")},
			nil,
			true},
		{Req{ID: "REQ-TEST-SWH-1", Title: "The control unit will calculate thrust.", Body: "It will also do much more."},
			ReqFilter{TitleRegexp: regexp.MustCompile("thrust")},
			nil,
			true},
		{Req{ID: "REQ-TEST-SWH-1", Title: "The control unit will calculate vertical take off speed.", Body: "It will also output thrust."},
			ReqFilter{TitleRegexp: regexp.MustCompile("thrust")},
			nil,
			false},
		{Req{ID: "REQ-TEST-SWH-1", Body: "thrust control"},
			ReqFilter{BodyRegexp: regexp.MustCompile("thrust")},
			nil,
			true},
		{Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			ReqFilter{IDRegexp: regexp.MustCompile("REQ-*"), BodyRegexp: regexp.MustCompile("thrust")},
			nil,
			true},
		{Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			ReqFilter{IDRegexp: regexp.MustCompile("REQ-DDLN-*"), BodyRegexp: regexp.MustCompile("thrust")},
			nil,
			false},

		// filter attributes
		{Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			ReqFilter{AnyAttributeRegexp: regexp.MustCompile("Demo*")},
			nil,
			true},
		{Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			ReqFilter{AnyAttributeRegexp: regexp.MustCompile("Test*")},
			nil,
			false},
		{Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Demo*")}},
			nil,
			true},
		{Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Color": "Brown"}},
			ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Demo*")}},
			nil,
			false},
		{Req{ID: "REQ-TEST-SWL-14", Attributes: map[string]string{"Verification": "Demonstration"}},
			ReqFilter{AttributeRegexp: map[string]*regexp.Regexp{"Verification": regexp.MustCompile("Test*")}},
			nil,
			false},

		// diffs
		{Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			ReqFilter{},
			map[string][]string{"REQ-TEST-SWL-1": make([]string, 0)},
			false},
		{Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			ReqFilter{},
			map[string][]string{"REQ-TEST-SWL-14": make([]string, 0)},
			true},
		{Req{ID: "REQ-TEST-SWL-14", Body: "thrust control"},
			ReqFilter{IDRegexp: regexp.MustCompile("X")},
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
