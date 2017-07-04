//@llr REQ-TRAQ-SWL-008
package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// ChangedSince produces a report of how requirments have changed between prg and this reqGraph
func (rg reqGraph) ChangedSince(prg reqGraph) (diffs map[string][]string) {
	if prg == nil {
		return
	}
	keys := map[string]bool{}
	for k, _ := range rg {
		keys[k] = true
	}
	for k, _ := range prg {
		keys[k] = true
	}
	var kk []string
	for k, _ := range keys {
		kk = append(kk, k)
	}
	sort.Strings(kk)
	diffs = make(map[string][]string)
	for _, k := range kk {
		if dd := rg[k].ChangedSince(prg[k]); dd != nil {
			diffs[k] = dd
		}
	}
	if len(diffs) == 0 {
		diffs = nil
	}
	return
}

func onlyLetters(s string) string {
	return strings.ToLower(strings.TrimFunc(s, func(r rune) bool { return !unicode.IsLetter(r) }))
}

// ChangedSince returns a set of messages that describe how r has changed
// from a previous version pr.
func (r *Req) ChangedSince(pr *Req) (diffs []string) {

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

	if r.Level != pr.Level {
		diffs = append(diffs, fmt.Sprintf("Level from %q to %q (should not happen!)", pr.Level, r.Level))
	}

	if c, p := onlyLetters(r.Title), onlyLetters(pr.Title); c != p {
		diffs = append(diffs, fmt.Sprintf("Changed from %q to %q", p, c))
	} else {
		// only bother with the bodies if the titles are the same
		// compare modulo spaces and punctuation, ie only the letters

		if onlyLetters(string(r.Body)) != onlyLetters(string(pr.Body)) {
			diffs = append(diffs, fmt.Sprintf("Body changed"))
		}
	}

	if r.Path != pr.Path {
		diffs = append(diffs, fmt.Sprintf("File in which found changed from %q to %q", pr.Path, r.Path))
	} else if r.FileHash != pr.FileHash {
		diffs = append(diffs, fmt.Sprintf("File %q contents changed", r.Path))
	}

	var keys []string
	for k, _ := range pr.Attributes {
		keys = append(keys, k)
	}
	for k, _ := range r.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
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
