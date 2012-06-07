// Diff Match and Patch – cleanup functions
// 	Original work: Copyright 2006 Google Inc.
// 	Go port:	Copyright 2012 M. Teichgräber
//
// Use of this source code is governed by the Apache License,
// Version 2.0, that can be found in the LICENSE file.

package dmp

import (
	"regexp"
	"strings"
	"unicode"
)

// Reduce the number of edits by eliminating semantically trivial equalities.
func (pdiffs *Diffs) CleanupSemantic() (diffs Diffs) {
	diffs = *pdiffs
	if len(diffs) == 0 {
		return
	}
	var (
		changes = false

		lenLastEquality int
		iLast           = -1

		// Number of bytes that changed prior to the equality
		lenIns1, lenDel1 int
		// Number of bytes that changed after the equality
		lenIns2, lenDel2 int
	)

	lookupPrevEquality := func(i int) int {
		for {
			i--
			if i < 0 || diffs[i].Op == Equal {
				break
			}
		}
		return i
	}

	for i := 0; i < len(diffs); i++ {
		d := diffs[i]
		if d.Op == Equal {
			// Equality found.
			lenIns1, lenDel1 = lenIns2, lenDel2
			lenIns2, lenDel2 = 0, 0
			lenLastEquality = len(d.Text)
			iLast = i
		} else {
			// An insertion or deletion
			switch d.Op {
			case Insert:
				lenIns2 += len(d.Text)
			case Delete:
				lenDel2 += len(d.Text)
			case deleteInsert:
				lenIns2 += len(d.Text)
				lenDel2 += len(d.Text)
			}

			// Eliminate an equality that is smaller or
			// equal to the edits on both sides of it.
			if iLast != -1 &&
				lenLastEquality <= max(lenIns1, lenDel1) &&
				lenLastEquality <= max(lenIns2, lenDel2) {

				// Walk back to offending equality.
				i = lookupPrevEquality(i)

				// Replace equality with a delete, and a corresponding an insert
				diffs[i].Op = deleteInsert

				iLast = -1
				i = lookupPrevEquality(i)
				if i != -1 && diffs[i].Op == Equal {
					// throw it away (needs to be reevaluated)
					i = lookupPrevEquality(i)
				}

				lenIns1, lenIns2 = 0, 0 // Reset the counters.
				lenDel1, lenDel2 = 0, 0
				changes = true
			}
		}
	}

	// Normalize the diff
	if changes {
		diffs.CleanupMerge()
	}
	diffs.CleanupSemanticLossless()

	// Find any overlaps between deletions and insertions:
	// 	e.g.: <del>abcxxx</del><ins>xxxdef</ins>
	//		-> <del>abc</del>xxx<ins>def</ins>
	//	e.g.: <del>xxxabc</del><ins>defxxx</ins>
	//		-> <ins>def</ins>xxx<del>abc</del>
	// Only extract an overlap if it is as big as the edit ahead or behind it.

	var wdiffs Diffs

	w := func(op int, text string) {} // dummy

	i := 0
	wPrev := func(op int, text string) {
		if wdiffs == nil {
			wdiffs = make(Diffs, i, len(diffs)*3/2)
			copy(wdiffs, diffs)
			w = func(op int, text string) {
				wdiffs = append(wdiffs, Diff{op, text})
			}
		}
		wdiffs = wdiffs[:len(wdiffs)-1]
		w(op, text)
	}

	for i = 1; i < len(diffs); i++ {
		d := diffs[i]
		if diffs[i-1].Op == Delete && d.Op == Insert {

			deletion := diffs[i-1].Text
			insertion := d.Text
			overlap1 := commonOverlap(deletion, insertion)
			overlap2 := commonOverlap(insertion, deletion)

			if oLen := len(overlap1); oLen >= len(overlap2) {
				if 2*oLen >= runeCount(deletion) || 2*oLen >= runeCount(insertion) {
					// Overlap found. Insert an equality and trim the surrounding edits.
					wPrev(Delete, deletion[:len(deletion)-oLen])
					w(Equal, overlap1)
					w(Insert, insertion[oLen:])
					continue
				}
			} else if oLen := len(overlap2); 2*oLen >= runeCount(deletion) || 2*oLen >= runeCount(insertion) {
				// Reverse overlap found.

				// Insert an equality and swap and trim the surrounding edits.
				wPrev(Insert, insertion[:len(insertion)-oLen])
				w(Equal, overlap2)
				w(Delete, deletion[oLen:])
				continue
			}
		}
		w(d.Op, d.Text)
	}
	if wdiffs != nil {
		diffs = wdiffs
	}
	*pdiffs = diffs
	return
}

// Look for single edits surrounded on both sides by equalities
// which can be shifted sideways to align the edit to a word boundary.
//	e.g.: The c<ins>at c</ins>ame. -> The <ins>cat </ins>came.
func (pDiffs *Diffs) CleanupSemanticLossless() (diffs Diffs) {
	diffs = *pDiffs

	var best fit
	iw := 1
	w := func(d Diff) {
		diffs[iw] = d
		iw++
	}
	inext := 2
	N := len(diffs)
	if N == 0 {
		return
	}
	for i := 1; i < N; i, inext = inext, inext+1 {
		d := diffs[i]
		if inext >= N {
			w(d)
			break
		}
		prev, next := &diffs[iw-1], &diffs[inext]
		if prev.Op != Equal || next.Op != Equal {
			w(d)
			continue
		}

		// This is a single edit surrounded by equalities
		cur := fit{
			equality1: prev.Text,
			edit:      d.Text,
			equality2: next.Text,
		}
		cur.shiftLeft()
		best = cur.shiftRight()

		if prev.Text != best.equality1 {
			// We have an improvement, save it back to the diff
			if best.equality1 != "" {
				prev.Text = best.equality1
			} else {
				iw-- // remove prev
			}
			d.Text = best.edit
			if best.equality2 != "" {
				next.Text = best.equality2
			} else {
				inext++ // remove next
			}
		}
		w(d)
	}
	diffs = diffs[:iw]
	*pDiffs = diffs
	return
}

type fit struct {
	equality1, edit, equality2 string
	score                      int
}

func (f *fit) calcScore() int {
	f.score = semanticScore(f.equality1, f.edit) + semanticScore(f.edit, f.equality2)
	return f.score
}

// shift the edit as far left as possible
func (f *fit) shiftLeft() {
	if cs := commonSuffix(f.equality1, f.edit); cs != "" {
		n := len(cs)
		f.equality1 = f.equality1[:len(f.equality1)-n]
		f.edit = cs + f.edit[:len(f.edit)-n]
		f.equality2 = cs + f.equality2
	}
}

// step character by character right, looking for the best fit
func (f *fit) shiftRight() (best fit) {
	best = *f
	best.calcScore()
	for f.edit != "" && f.equality2 != "" && firstRune(f.edit) == firstRune(f.equality2) {
		f.equality1 += firstUTF8(f.edit)
		f.edit = f.edit[1:] + firstUTF8(f.equality2)
		f.equality2 = f.equality2[1:]
		f.calcScore()

		// The >= encourages trailing rather than leading whitespace on edits
		if f.score >= best.score {
			best = *f
		}
	}
	return
}

// Given two strings, compute a score representing whether the internal
// boundary falls on logical boundaries
// Scores range from 6 (best) to 0 (worst).
func semanticScore(one, two string) (score int) {
	if one == "" || two == "" {
		// Edges are the best
		return 6
	}

	// Each port of this function behaves slightly differently due to
	// subtle differences in each language's definition of things like
	// 'whitespace'. Since this function's purpose is largely cosmetic,
	// the choice has been made to use each language's native features
	// rather than force total conformity.
	r1 := lastRune(one)
	r2 := firstRune(two)
	nonAlphaNum1 := !unicode.IsLetter(r1) && !unicode.IsDigit(r1)
	nonAlphaNum2 := !unicode.IsLetter(r2) && !unicode.IsDigit(r2)
	space1 := nonAlphaNum1 && unicode.IsSpace(r1)
	space2 := nonAlphaNum2 && unicode.IsSpace(r2)
	lineBreak1 := space1 && unicode.IsControl(r1)
	lineBreak2 := space2 && unicode.IsControl(r2)
	blankLine1 := lineBreak1 && blankLineEnd.MatchString(one)
	blankLine2 := lineBreak2 && blankLineStart.MatchString(two)

	switch {
	case blankLine1 || blankLine2:
		// blank lines
		score = 5
	case lineBreak1 || lineBreak2:
		// line breaks
		score = 4
	case nonAlphaNum1 && !space1 && space2:
		// end of sentences
		score = 3
	case space1 || space2:
		// whitespace
		score = 2
	case nonAlphaNum1 || nonAlphaNum2:
		// non-alphanumeric
		score = 1
	}
	return
}

// Define some regex patterns for matching boundaries
var (
	blankLineEnd   = regexp.MustCompile(`(?s)\n\r?\n(\z|\r?\n\z)`)
	blankLineStart = regexp.MustCompile(`(?s)\A\r?\n\r?\n`)
)

// Reduce the number of edits by eliminating operationally trivial equalities.
func (pDiffs *Diffs) CleanupEfficiency(editCost int) (diffs Diffs) {
	diffs = *pDiffs
	if len(diffs) == 0 {
		return
	}

	if editCost == 0 {
		editCost = DefaultEditCost
	}

	var (
		changes = false

		// are there certain operations before, or after the last equality?
		preIns, preDel   bool
		postIns, postDel bool

		iLast = -1 // index of last equality
		iSafe = -1 // index of last Diff that is known to be unsplitable.
	)

	v := func(v bool) int {
		if v {
			return 1
		}
		return 0
	}

	lookupPrevEquality := func(i int) int {
		for {
			i--
			if i == iSafe || diffs[i].Op == Equal {
				break
			}
		}
		return i
	}

	for i := 0; i < len(diffs); i++ {
		d := &diffs[i]
		if d.Op == Equal {
			// Equality found
			if !exceedsRuneCount(d.Text, editCost-1) && (postIns || postDel) {
				// Candidate found
				preIns, preDel = postIns, postDel
				iLast = i
			} else {
				// Not a candidate, and can never become one.
				iLast = -1
				iSafe = i
			}
			postIns, postDel = false, false
		} else {
			// An insertion or deletion
			switch d.Op {
			case Delete:
				postDel = true
			case Insert:
				postIns = true
			case deleteInsert:
				postDel, postIns = true, true
			}

			// Five types to be split:
			// <ins>A</ins><del>B</del>XY<ins>C</ins><del>D</del>
			// <ins>A</ins>X<ins>C</ins><del>D</del>
			// <ins>A</ins><del>B</del>X<ins>C</ins>
			// <del>A</del>X<ins>C</ins><del>D</del>
			// <ins>A</ins><del>B</del>X<del>C</del>
			//
			if iLast != -1 &&
				((preIns && preDel && postIns && postDel) ||
					(!exceedsRuneCount(diffs[iLast].Text, editCost/2-1) &&
						v(preIns)+v(preDel)+v(postIns)+v(postDel) == 3)) {
				// Walk back to offending equality.
				i = lookupPrevEquality(i)

				// Replace equality with a temporary deleteInsert, that will
				// be resolved by cleanupMerge
				diffs[i].Op = deleteInsert
				iLast = -1
				if preIns && preDel {
					// No changes made which could affect previous entry, keep going.
					postIns, postDel = true, true
					iSafe = i - 1
				} else {
					i = lookupPrevEquality(i)
					if i != -1 && diffs[i].Op == Equal {
						// throw it away (needs to be reevaluated)
						if i != iSafe {
							i = lookupPrevEquality(i)
						}
					}
					postIns, postDel = false, false
				}
				changes = true
			}
		}
	}

	if changes {
		diffs.CleanupMerge()
	}
	*pDiffs = diffs
	return
}

// Reorder and merge like edit sections.  Merge equalities.
// Any edit section can move as long as it doesn't cross an equality.
func (pDiffs *Diffs) CleanupMerge() (diffs Diffs) {
	var insBuf, delBuf strbuf
	diffs = append(*pDiffs, Diff{Equal, ""})
	iw := 0 // write index

	w := func(op int, text string) {
		diffs[iw] = Diff{op, text}
		iw++
	}
	prevEqual := func() (eq *Diff) {
		if iw == 0 {
			return
		}
		if p := &diffs[iw-1]; p.Op == Equal {
			eq = p
		}
		return
	}

	for _, d := range diffs {
		switch d.Op {
		case noop:
		case Insert:
			insBuf = append(insBuf, d.Text)
		case Delete:
			delBuf = append(delBuf, d.Text)
		case deleteInsert:
			insBuf = append(insBuf, d.Text)
			delBuf = append(delBuf, d.Text)
		case Equal:
			textIns := insBuf.join()
			textDel := delBuf.join()
			if textDel != "" && textIns != "" { // both types
				// Factor out any common prefixes
				if pfx := commonPrefix(textIns, textDel); pfx != "" {
					if iw != 0 {
						if prev := prevEqual(); prev == nil {
							panic("Previous diff should have been an equality")
						} else {
							prev.Text += pfx
						}
					} else {
						w(Equal, pfx)
					}
					textIns = textIns[len(pfx):]
					textDel = textDel[len(pfx):]
				}
				// Factor out any common suffixies.
				if sfx := commonSuffix(textIns, textDel); sfx != "" {
					d.Text = sfx + d.Text
					textIns = textIns[:len(textIns)-len(sfx)]
					textDel = textDel[:len(textDel)-len(sfx)]
				}
			}
			// Insert the merged records.
			if textDel != "" {
				w(Delete, textDel)
			}
			if textIns != "" {
				w(Insert, textIns)
			}
			if prev := prevEqual(); prev != nil {
				prev.Text += d.Text
			} else {
				w(Equal, d.Text)
			}
		}
	}
	diffs = diffs[:iw]
	if last := len(diffs) - 1; diffs[last].Text == "" {
		diffs = diffs[:last]
	}

	// Second pass: look for single edits surrounded on both sides by equalities
	// which can be shifted sideways to eliminate an equality.
	// e.g: A<ins>BA</ins>C -> <ins>AB</ins>AC
	changes := false
	iLast := len(diffs) - 1
	for i, d := range diffs {
		if i == 0 || i == iLast {
			continue
		}
		prev, next := &diffs[i-1], &diffs[i+1]
		if prev.Op != Equal || next.Op != Equal {
			continue
		}
		// This is a single edit surrounded by equalities.
		if strings.HasSuffix(d.Text, prev.Text) {
			diffs[i].Text = prev.Text + d.Text[:len(d.Text)-len(prev.Text)]
			next.Text = prev.Text + next.Text
			prev.Op = noop
			changes = true
		} else if strings.HasPrefix(d.Text, next.Text) {
			prev.Text += next.Text
			diffs[i].Text = d.Text[len(next.Text):] + next.Text
			next.Op = noop
			changes = true
		}
	}

	// If shifts were made, the diff needs reordering and another shift sweep
	if changes {
		diffs.CleanupMerge()
	}
	*pDiffs = diffs
	return diffs
}
