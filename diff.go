// Diff Match and Patch – diff main functions
// 	Original work: Copyright 2006 Google Inc.
// 	Go port:	Copyright 2012 M. Teichgräber
//
// Use of this source code is governed by the Apache License,
// Version 2.0, that can be found in the LICENSE file.

package dmp

import (
	"fmt"
	. "github.com/knieriem/dmp/rstring"
	"strings"
	"time"
)

const (
	DefaultEditCost = 4
	DefaultTimeout  = time.Second
	NoTimeout       = -1
)

// The data structure representing a diff is a slice of Diff objects:
//	[]Diff{ {Delete, "Hello"}, {Insert, "Goodbye"}, {Equal, " world."}}
// which means: delete "Hello", add "Goodbye" and keep " world."
type Diffs []Diff

type Diff struct {
	Op   int
	Text string
}

const (
	// Operations
	noop         = 0
	Equal        = '='
	Insert       = '+'
	Delete       = '-'
	deleteInsert = '±'
)

func (d Diff) String() string {
	return fmt.Sprintf("%c<%s> ", d.Op, d.Text)
}

func (d *Diffs) add(op int, text string) {
	*d = append(*d, Diff{op, text})
}

type differ struct {
	Diffs
	checkLines bool
	deadLine   time.Time
	bisectV    []int
}

// Find the differences between two texts.
// If checkLines is false, then don't run a line-level
// diff first to identify the changed areas. If it is true,
// then run a faster slightly less optimal diff.
//
// If timeout is NoTimeout, or -1, it timeout will be inactive.
// If it is 0, DefaultTimeout will be used.
func DiffMain(text1, text2 string, checkLines bool, timeout time.Duration) Diffs {
	d := new(differ)
	if timeout != -1 {
		if timeout == 0 {
			timeout = DefaultTimeout
		}
		d.deadLine = time.Now().Add(timeout)
	}
	d.diffMain(text1, text2, checkLines)
	d.CleanupMerge()
	return d.Diffs
}

// Find the differences between two texts.  Simplifies the problem by
// stripping any common prefix or suffix off the texts before diffing.
func (d *differ) diffMain(text1, text2 string, checkLines bool) {
	if text1 == text2 {
		if text1 != "" {
			d.add(Equal, text1)
		}
		return
	}

	// Trim off common prefix (speedup)
	commonPfx := commonPrefix(text1, text2)
	text1 = text1[len(commonPfx):]
	text2 = text2[len(commonPfx):]

	// Trim off common suffix (speedup)
	commonSfx := commonSuffix(text1, text2)
	text1 = text1[:len(text1)-len(commonSfx)]
	text2 = text2[:len(text2)-len(commonSfx)]

	// Restore the prefix
	if commonPfx != "" {
		d.add(Equal, commonPfx)
	}

	// Compute the diff on the middle block
	d.compute(LRString(text1), LRString(text2), checkLines)

	// Restore the suffix
	if commonSfx != "" {
		d.add(Equal, commonSfx)
	}
	return
}

// Find the differences between two texts.  Assumes that the texts do not
// have any common prefix or suffix.
func (d *differ) compute(text1, text2 LRstring, checkLines bool) {
	if text1.String() == "" {
		// Just add some text (speedup)
		d.add(Insert, text2.String())
		return
	}

	if text2.String() == "" {
		// Just delete some text (speedup)
		d.add(Delete, text1.String())
		return
	}

	var long, short string
	var op int

	if text1.Count() > text2.Count() {
		long, short = text1.String(), text2.String()
		op = Delete
	} else {
		long, short = text2.String(), text1.String()
		op = Insert
	}
	if i := strings.Index(long, short); i != -1 {
		// Shorter text is inside the longer text (speedup).
		d.add(op, long[0:i])
		d.add(Equal, short)
		d.add(op, long[i+len(short):])
		return
	}

	if !exceedsRuneCount(short, 1) {
		// After the previous speedup, the character can't be an equality.
		d.add(Delete, text1.String())
		d.add(Insert, text2.String())
		return
	}

	// Check to see if the problem can be split in two.
	if hm := findHalfMatch(text1, text2, d.deadLine.IsZero()); hm != nil {
		// Send both pairs off for separate processing, and merge the results.
		d.diffMain(hm.prefix1, hm.prefix2, checkLines)
		d.add(Equal, hm.common.String())
		d.diffMain(hm.suffix1, hm.suffix2, checkLines)
		return
	}

	if checkLines && text1.Count() > 100 && text2.Count() > 100 {
		d.diffLineMode(text1.String(), text2.String())
	} else {
		var s1, s2 IRstring
		d.bisect(s1.Init(text1.String()), s2.Init(text2.String()))
	}
	return
}

func (d *differ) diffLineMode(text1, text2 string) {
	// Scan the text on a line-by-line basis first.
	b := diffLinesToChars(text1, text2)

	ld := *d
	ld.Diffs = nil
	ld.diffMain(b.chars1, b.chars2, false)

	// Convert the diff back to original text.
	diffCharsToLines(ld.Diffs, b.lines)
	// Eliminate freak matches (e.g. blank lines)
	ld.CleanupSemantic()

	// Rediff any replacement blocks, this time character-by-character.
	ld.add(Equal, "")
	var textDel, textIns string
	for i, diff := range ld.Diffs {
		switch diff.Op {
		case Insert:
			textIns += diff.Text
		case Delete:
			textDel += diff.Text
		case Equal:
			// Upon reaching an equality, check for prior redundancies.
			switch {
			case textDel != "" && textIns != "":
				d.diffMain(textDel, textIns, false)
			case textDel != "":
				d.add(Delete, textDel)
			case textIns != "":
				d.add(Insert, textIns)
			}
			if i+1 != len(ld.Diffs) {
				d.add(Equal, diff.Text)
			}
			textDel = ""
			textIns = ""
		}
	}
}

// Find the 'middle snake' of a diff, split the problem in two
// and return the recursively constructed diff.
// See Myers 1986 paper: An O(ND) Difference Algorithm and Its Variations.
func (d *differ) bisect(text1, text2 *IRstring) {
	// Cache the text lengths to prevent multiple calls
	text1Len, text2Len := text1.Count(), text2.Count()
	maxD := (text1Len + text2Len + 1) / 2
	vOff := maxD
	vLen := 2 * maxD
	if cap(d.bisectV) < vLen*2 {
		d.bisectV = make([]int, vLen*2)
	}
	v1 := d.bisectV[:vLen]
	v2 := d.bisectV[vLen : 2*vLen]
	for x := range v1 {
		v1[x] = -1
		v2[x] = -1
	}
	v1[vOff+1] = 0
	v2[vOff+1] = 0
	Δ := text1Len - text2Len

	// If the total number of characters is odd, then the front path will
	// collide with the reverse path.
	front := isOdd(Δ)

	// Offsets for start and end of k loop.
	// Prevents mapping of space beyond the grid.
	k1start := 0
	k1end := 0
	k2start := 0
	k2end := 0

	var x1, y1, k1off int
	var x2, y2, k2off int

	for D := 0; D < maxD; D++ {
		if !d.deadLine.IsZero() {
			if time.Now().After(d.deadLine) {
				break
			}
		}

		// Walk the front path one step
		for k1 := -D + k1start; k1 <= D-k1end; k1 += 2 {
			k1off = vOff + k1
			x1 = 0
			if k1 == -D || (k1 != D && v1[k1off-1] < v1[k1off+1]) {
				x1 = v1[k1off+1]
			} else {
				x1 = v1[k1off-1] + 1
			}
			y1 = x1 - k1
			for x1 < text1Len && y1 < text2Len && text1.At(x1) == text2.At(y1) {
				x1++
				y1++
			}
			v1[k1off] = x1

			switch {
			case x1 > text1Len:
				// Ran off the right of the graph.
				k1end += 2
			case y1 > text2Len:
				// Ran off the bottom of the graph.
				k1start += 2
			case front:
				k2off = vOff + Δ - k1
				if k2off >= 0 && k2off < vLen && v2[k2off] != -1 {
					// Mirror x2 onto top-left coordinate system.
					x2 = text1Len - v2[k2off]
					if x1 >= x2 {
						// Overlap detected
						d.bisectSplit(text1, text2, x1, y1)
						return
					}
				}
			}
		}

		// Walk the reverse path one step
		for k2 := -D + k2start; k2 <= D-k2end; k2 += 2 {
			k2off = vOff + k2
			x2 = 0
			if k2 == -D ||
				(k2 != D && v2[k2off-1] < v2[k2off+1]) {
				x2 = v2[k2off+1]
			} else {
				x2 = v2[k2off-1] + 1
			}
			y2 = x2 - k2
			for x2 < text1Len && y2 < text2Len && text1.At(text1Len-x2-1) == text2.At(text2Len-y2-1) {
				x2++
				y2++
			}
			v2[k2off] = x2

			switch {

			case x2 > text1Len:
				// Ran off the left of the graph.
				k2end += 2
			case y2 > text2Len:
				// Ran off the top of the graph.
				k2start += 2
			case !front:
				k1off = vOff + Δ - k2
				if k1off >= 0 && k1off < vLen && v1[k1off] != -1 {
					x1 = v1[k1off]
					y1 = vOff + x1 - k1off
					// Mirror x2 onto top-left coordinate system.
					x2 = text1Len - x2
					if x1 >= x2 {
						// Overlap detected.
						d.bisectSplit(text1, text2, x1, y1)
						return
					}
				}
			}
		}
	}

	// Diff took too long and hit the deadline or
	// number of diffs equals number of characters,
	// no commonality at all.
	d.add(Delete, text1.String())
	d.add(Insert, text2.String())
	return
}

// Given the location of the `middle snake', split the diff in two parts
// and recurse.
//	x: Index of split point in text1.
// 	y: Index of split point in text2.
func (d *differ) bisectSplit(text1, text2 *IRstring, x, y int) {
	s1, i1 := text1.String(), text1.BytePos(x)
	s2, i2 := text2.String(), text2.BytePos(y)

	// Compute both diffs serially.
	d.diffMain(s1[:i1], s2[:i2], false)
	d.diffMain(s1[i1:], s2[i2:], false)
	return
}
