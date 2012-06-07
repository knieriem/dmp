// Diff Match and Patch – halfmatch speedup
// 	Original work: Copyright 2006 Google Inc.
// 	Go port:	Copyright 2012 M. Teichgräber
//
// Use of this source code is governed by the Apache License,
// Version 2.0, that can be found in the LICENSE file.

package dmp

import (
	. "github.com/knieriem/dmp/rstring"
	"strings"
)

type halfMatch struct {
	prefix1 string
	suffix1 string
	prefix2 string
	suffix2 string
	common  LRstring
}

// Do the two texts share a substring which is at least half the length of
// the longer text?
// This speedup can produce non-minimal diffs.
// Returns a *halfMatch, containing the prefix of text1, the
// suffix of text1, the prefix of text2, the suffix of text2 and the
// common middle.  Or nil if there was no match.
func findHalfMatch(text1, text2 LRstring, unlimitedTime bool) (hm *halfMatch) {
	if unlimitedTime {
		// Don't risk returning a non-optimal diff
		return
	}

	var long, short LRstring
	if text1.Count() > text2.Count() {
		long, short = text1, text2
	} else {
		long, short = text2, text1
	}
	if long.Count() < 4 || short.Count()*2 < long.Count() {
		return // Pointless
	}

	// First check if the second quarter is the seed for a half-match
	hm1 := findHalfMatchAroundIndex(long, short.String(), (long.Count()+3)/4)

	// Check again based on the third quarter
	hm2 := findHalfMatchAroundIndex(long, short.String(), (long.Count()+1)/2)

	switch {
	case hm1 == nil && hm2 == nil:
		return
	case hm2 == nil:
		hm = hm1
	case hm1 == nil:
		hm = hm2
	// Both matched.  Select the longest
	case hm1.common.Count() > hm2.common.Count():
		hm = hm1
	default:
		hm = hm2
	}

	// A half-match was found, sort out the return data
	if text1.Count() <= text2.Count() {
		hm = &halfMatch{hm.prefix2, hm.suffix2, hm.prefix1, hm.suffix1, hm.common}
	}
	return
}

// Does a substring of shorttext exist within longtext such that the
// substring is at least half the length of longtext?
func findHalfMatchAroundIndex(longText LRstring, short string, i0 int) (hm *halfMatch) {
	var best halfMatch

	// Start with a 1/4 length substring at position i0 as a seed.
	i0, iEnd := longText.ByteIndices(i0, i0+longText.Count()/4)
	long := longText.String()
	seed := long[i0:iEnd]
	j := -1
	for {
		if iSeed := strings.Index(short[j+1:], seed); iSeed == -1 {
			break
		} else {
			j += 1 + iSeed
		}
		prefix := LRString(commonPrefix(long[i0:], short[j:]))
		suffix := LRString(commonSuffix(long[:i0], short[:j]))

		if best.common.Count() < suffix.Count()+prefix.Count() {
			best.common.Concat(suffix, prefix)
			best.prefix1 = long[:i0-suffix.Len()]
			best.suffix1 = long[i0+prefix.Len():]
			best.prefix2 = short[:j-suffix.Len()]
			best.suffix2 = short[j+prefix.Len():]
		}
	}
	if best.common.Count()*2 >= longText.Count() {
		hm = &best
	}
	return
}
