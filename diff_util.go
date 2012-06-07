// Diff Match and Patch – Diffs methods
// 	Original work: Copyright 2006 Google Inc.
// 	Go port:	Copyright 2012 M. Teichgräber
//
// Use of this source code is governed by the Apache License,
// Version 2.0, that can be found in the LICENSE file.

package dmp

import (
	. "github.com/knieriem/dmp/rstring"
	"html"
	"strings"
)

// Loc1 is a location in text1; compute and return the equivalent location in
// text2.
//	e.g. "The cat" vs "The big cat", 1->1, 5->8
func (diffs Diffs) XIndex(loc1 int) (loc2 int) {
	var lastDiff Diff

	chars1, chars2 := 0, 0
	lastChars1, lastChars2 := 0, 0

	for _, d := range diffs {
		if d.Op != Insert {
			// Equality or deletion
			chars1 += len(d.Text)
		}
		if d.Op != Delete {
			// Equality or insertion
			chars2 += len(d.Text)
		}
		if chars1 > loc1 {
			// Overshot the location
			lastDiff = d
			break
		}
		lastChars1, lastChars2 = chars1, chars2
	}
	if lastDiff.Op == Delete {
		// The location was deleted
		loc2 = lastChars2
	} else {
		// Add the remaining character length
		loc2 = lastChars2 + (loc1 - lastChars1)
	}
	return
}

// Convert a Diff list into a pretty HTML report.
func (diffs Diffs) PrettyHTML() (s string) {
	r := strings.NewReplacer("\n", "&para;<br>")
	for _, d := range diffs {
		text := html.EscapeString(string(d.Text))
		text = r.Replace(text)
		switch d.Op {
		case Insert:
			s += `<ins style="background:#e6ffe6;">` + text + "</ins>"
		case Delete:
			s += `<del style="background:#ffe6e6;">` + text + "</del>"
		case Equal:
			s += "<span>" + text + "</span>"
		}
	}
	return
}

// Compute and return the source text (all equalities and deletions).
func (diffs Diffs) Text1() (source string) {
	for _, d := range diffs {
		if d.Op != Insert {
			source += d.Text
		}
	}
	return
}

// Compute and return the destination text (all equalities and insertions).
func (diffs Diffs) Text2() (dest string) {
	for _, d := range diffs {
		if d.Op != Delete {
			dest += d.Text
		}
	}
	return
}

// Compute the Levenshtein distance – the number of inserted, deleted or
// substituted characters.
func (diffs Diffs) Levenshtein() (levenshtein int) {
	insertions := 0
	deletions := 0
	for _, d := range diffs {
		switch d.Op {
		case Insert:
			insertions += Rstring(d.Text).Count()
		case Delete:
			deletions += Rstring(d.Text).Count()
			break
		case Equal:
			// A deletion and an insertion is one substitution
			levenshtein += max(insertions, deletions)
			insertions = 0
			deletions = 0
		}
	}
	levenshtein += max(insertions, deletions)
	return
}
