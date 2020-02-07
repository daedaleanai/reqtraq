package main

import (
	"io/ioutil"
	"regexp"
	"testing"
)

func TestReports(t *testing.T) {
	rg, err := CreateReqGraph(*fCertdocPath, *fCodePath)
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

func checkFilteredReports(t *testing.T, rg *reqGraph, filter *ReqFilter) {
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
