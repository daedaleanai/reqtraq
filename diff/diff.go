/*
Functions which compare two requirements graphs and return a map-of-slice-of-strings structure which describe how they differ.
*/

package diff

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/daedaleanai/reqtraq/reqs"
)

// ChangedSince produces a report of how requirements have changed between two requirement graphs
// @llr REQ-TRAQ-SWL-18
func ChangedSince(rg, prg *reqs.ReqGraph) (diffs map[string][]string) {
	if prg == nil {
		return
	}
	keys := map[string]bool{}
	for k := range rg.Reqs {
		keys[k] = true
	}
	for k := range prg.Reqs {
		keys[k] = true
	}
	var kk []string
	for k := range keys {
		kk = append(kk, k)
	}
	sort.Strings(kk)
	diffs = make(map[string][]string)
	for _, k := range kk {
		if dd := changedSince(rg.Reqs[k], prg.Reqs[k]); dd != nil {
			diffs[k] = dd
		}
	}
	if len(diffs) == 0 {
		diffs = nil
	}
	fmt.Printf("%v\n", diffs)
	return
}

// changedSince returns a set of messages that describe how a requirement has changed from a previous version.
// @llr REQ-TRAQ-SWL-18, REQ-TRAQ-SWL-40
func changedSince(r, pr *reqs.Req) (diffs []string) {
	if r == nil && pr == nil {
		return nil
	}

	if r == nil {
		return []string{"MISSING"}
	}

	if pr == nil {
		return []string{"ADDED"}
	}

	if rd, prd := r.IsDeleted(), pr.IsDeleted(); rd || prd {
		if rd && prd {
			return nil
		}
		if rd {
			return []string{"DELETED"}
		}
		if rd {
			return []string{"UNDELETED"}
			// no point in comparing all the other attributes
		}
	}

	// r and pr exist and are not deleted; more in-depth compares

	if r.ID != pr.ID {
		diffs = append(diffs, fmt.Sprintf("ID from %q to %q (should not happen!)", pr.ID, r.ID))
	}

	if c, p := onlyLetters(r.Title), onlyLetters(pr.Title); c != p {
		diffs = append(diffs, fmt.Sprintf("Changed from %q to %q", p, c))
	} else {
		// only bother with the bodies if the titles are the same
		// compare modulo spaces and punctuation, ie only the letters

		if onlyLetters(r.Body) != onlyLetters(pr.Body) {
			diffs = append(diffs, fmt.Sprintf("Body changed"))
		}
	}

	if r.RepoName != pr.RepoName || r.Document.Path != pr.Document.Path {
		diffs = append(diffs, fmt.Sprintf("File in which found changed from (%q - %q) to (%q - %q)",
			pr.RepoName, pr.Document.Path, r.RepoName, r.Document.Path))
	}

	var keys []string
	for k := range pr.Attributes {
		keys = append(keys, k)
	}
	for k := range r.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if k == "PARENTS" {
			// done later
			continue
		}
		v, ok := r.Attributes[k]
		pv, pok := pr.Attributes[k]
		// at least one of them is there
		if ok != pok {
			if ok {
				diffs = append(diffs, fmt.Sprintf("Added %q: %q", k, v))
			} else {
				diffs = append(diffs, fmt.Sprintf("Removed %q", k))
			}
		} else if p, c := onlyLetters(v), onlyLetters(pv); p != c {
			diffs = append(diffs, fmt.Sprintf("Changed %q from %q to %q", k, p, c))
		}
	}

	// report changes in parent pointers.
	c, p := map[string]bool{}, map[string]bool{}
	for _, v := range r.ParentIds {
		c[v] = true
	}
	for _, v := range pr.ParentIds {
		p[v] = true
		if !c[v] {
			diffs = append(diffs, fmt.Sprintf("Removed parent %q", v))
		}
	}
	for _, v := range r.ParentIds {
		if !p[v] {
			diffs = append(diffs, fmt.Sprintf("Added parent %q", v))
		}
	}

	return
}

// onlyLetters trims non letter characters from a string to enable meaningful comparison of requirements
// @llr REQ-TRAQ-SWL-40
func onlyLetters(s string) string {
	return strings.ToLower(strings.TrimFunc(s, func(r rune) bool { return !unicode.IsLetter(r) }))
}
