// Diff Match and Patch – utility functions
// 	Original work: Copyright 2006 Google Inc.
// 	Go port:	Copyright 2012 M. Teichgräber
//
// Use of this source code is governed by the Apache License,
// Version 2.0, that can be found in the LICENSE file.

package dmp

import (
	"strings"
	"unicode/utf8"
)

// Determine the common prefix of two strings.
// Returns the number of bytes common to the
// start of each string.
func commonPrefix(text1, text2 string) string {
	n := min(len(text1), len(text2))
	for i := 0; i < n; i++ {
		if text1[i] != text2[i] {
			n = i
			for n > 0 && !utf8.RuneStart(text1[n]) {
				n--
			}
			break
		}
	}
	return text1[:n]
}

// Determine the common suffix of two strings.
// Returns the number of bytes common to the end of
// each string.
func commonSuffix(text1, text2 string) string {
	n1 := len(text1)
	n2 := len(text2)
	n := min(n1, n2)

	for i := 1; i <= n; i++ {
		if text1[n1-i] != text2[n2-i] {
			n = i - 1
			for n > 0 && !utf8.RuneStart(text1[n1-n]) {
				n--
			}
			break
		}
	}
	return text1[n1-n:]
}

// Determine if the suffix of the first string is the prefix of the second.
// Returns the number of bytes common to the end of the first
// string and the start of the second string.
func commonOverlap(text1, text2 string) (best string) {
	// Cache the text lengths to prevent multiple calls.
	n1 := len(text1)
	n2 := len(text2)

	switch {
	// Eliminate the null case
	case n1 == 0 || n2 == 0:
		return

	// Truncate the longer string
	case n1 > n2:
		text1 = utf8SliceRightX(text1, n1-n2)
		n1 = len(text1)
	case n2 < n1:
		text2 = text2[:n1]
	}

	// Quick check for the worst case.
	if text1 == text2 {
		return text1
	}

	// Start by looking for a single character match
	// and increase length until no match is found.
	length := len(utf8SliceRight(text1, n1-1))
	for length <= n1 {
		pattern := text1[n1-length:]
		found := strings.Index(text2, pattern)
		if found == -1 {
			return
		}
		length += found
		if found == 0 {
			best = text2[:length]
			if length == n1 {
				return
			}
			length = len(utf8SliceRight(text1, n1-length-1))
		}
	}
	return
}

func min(n1, n2 int) int {
	if n1 < n2 {
		return n1
	}
	return n2
}

func max(n1, n2 int) int {
	if n1 > n2 {
		return n1
	}
	return n2
}

func isOdd(v int) bool {
	return v&1 != 0
}

// Create a slice s[i0:], taking care that i0
// points to the start of a character. Decrease
// i0 until this condition is met.
func utf8SliceRight(s string, i0 int) string {
	for i0 >= 0 && !utf8.RuneStart(s[i0]) {
		i0--
	}
	return s[i0:]
}

// Create a slice s[i0:], taking care that i0
// points to the start of a character. Increase
// i0 until this condition is met.
func utf8SliceRightX(s string, i0 int) string {
	n := len(s)
	for i0 != n && !utf8.RuneStart(s[i0]) {
		i0++
	}
	return s[i0:]
}

func firstRune(s string) (r rune) {
	r, _ = utf8.DecodeRuneInString(s)
	return
}

func lastRune(s string) (r rune) {
	r, _ = utf8.DecodeLastRuneInString(s)
	return
}

func firstUTF8(s string) string {
	_, n := utf8.DecodeRuneInString(s)
	return s[:n]
}

func lastUTF8(s string) string {
	_, n := utf8.DecodeLastRuneInString(s)
	return s[len(s)-n:]
}

func exceedsRuneCount(s string, count int) (exceeds bool) {
	for _ = range s {
		if count == 0 {
			exceeds = true
			break
		}
		count--
	}
	return
}

func runeCount(s string) int {
	return utf8.RuneCountInString(s)
}

type strbuf []string

func (p *strbuf) join() (s string) {
	buf := *p
	s = strings.Join(buf, "")
	*p = buf[:0]
	return
}
