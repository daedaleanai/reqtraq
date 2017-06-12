package main

import (
	"testing"
)

func TestReq_ReqType(t *testing.T) {
	req := Req{ID: "REQ-0-DDLN-SWL-001"}

	if v := req.ReqType(); v != "SWL" {
		t.Error("Expected SWL got", v)
	}

}

func TestReq_ReqTypeNoMatch(t *testing.T) {
	req := Req{ID: "Garbage"}

	if v := req.ReqType(); v != "" {
		t.Error("Expected nothing got", v)
	}

}
