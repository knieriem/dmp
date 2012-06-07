// Diff Match and Patch – line mode conversion utilities
// 	Original work: Copyright 2006 Google Inc.
// 	Go port:	Copyright 2012 M. Teichgräber
//
// Use of this source code is governed by the Apache License,
// Version 2.0, that can be found in the LICENSE file.

package dmp

import (
	"bytes"
	"fmt"
	"strings"
)

type linesDesc struct {
	chars1, chars2 string
	lines          []string
}

func (d *linesDesc) String() string {
	return fmt.Sprintf("#1:%q, #2:%q, lines:%q\n", d.chars1, d.chars2, d.lines)
}

// Split two texts into a list of strings. Reduce the texts to a string of
// hashes where each Unicode character represents one line.
// Returns a *linesDesc containing the encoded text1, the encoded text2
// and the list of unique strings. The zeroth element of the list of
// unique strings is intentionally blank.
func diffLinesToChars(text1, text2 string) *linesDesc {
	var d linesDesc
	m := newLineMunger()
	d.chars1 = m.linesToChars(text1)
	d.chars2 = m.linesToChars(text2)
	d.lines = m.lineArray
	return &d
}

type lineMunger struct {
	lineArray []string       // e.g. d.lines[4] == "Hello\n"
	lineHash  map[string]int // e.g. lineHash["Hello\n"] == 4
}

func newLineMunger() *lineMunger {
	var m lineMunger

	// "\x00" is a valid character, but various debuggers don't like it.
	// So we'll insert a junk entry to avoid generating a null character.
	m.lineArray = []string{""}
	m.lineHash = make(map[string]int, 16)
	return &m
}

// Split a text into a list of strings. Reduce the texts to a string of
// hashes where each Unicode character represents one line.
// Returns encoded string.
func (m *lineMunger) linesToChars(text string) string {
	lines := strings.SplitAfter(text, "\n")
	chars := bytes.NewBuffer(make([]byte, 0, 2*len(lines)))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if id, ok := m.lineHash[line]; ok {
			chars.WriteRune(rune(id))
		} else {
			m.lineArray = append(m.lineArray, line)
			id = len(m.lineArray) - 1
			m.lineHash[line] = id
			chars.WriteRune(rune(id))
		}
	}
	return chars.String()
}

// Rehydrate the text in a diff from a string of line hashes to
// real lines of text.
func diffCharsToLines(diffs []Diff, lines []string) {
	var b bytes.Buffer
	for i := range diffs {
		for _, r := range diffs[i].Text {
			b.WriteString(lines[r])
		}
		diffs[i].Text = b.String()
		b.Reset()
	}
}
