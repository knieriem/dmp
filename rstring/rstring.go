/*
Package rstring provides an efficient way to index strings by rune rather than by byte.

There are three types, Rstring, LRstring, and IRstring, that provide different
levels of buffering. For instance, LRstring buffers the rune count, so that it
does not have to be recomputed, while Rstring is just a wrapper around string:

 	R	LR	IR	Buffering
	-	+	+	RuneCount
	-	-	+	Position

The IRstring implementation is based on the standard library's exp/utf8string package.

*/
package rstring

import (
	"unicode/utf8"
)

type Rstring string

func (s Rstring) String() string {
	return string(s)
}
func (s Rstring) Len() int {
	return len(s)
}

func (s Rstring) Count() int {
	return utf8.RuneCountInString(string(s))
}

func (s Rstring) BytePos(runePos int) int {
	var n int

	for i := range s {
		if n == runePos {
			return i
		}
		n++
	}
	panic(outOfRange)
}

func (s Rstring) ByteIndices(start, end int) (i0, iEnd int) {
	var n int

	for i := range s {
		if n == start {
			i0 = i
		}
		if n == end {
			iEnd = i
			break
		}
		n++
	}
	return
}

type LRstring struct {
	Rstring
	count int
}

func LRString(s string) (lrstr LRstring) {
	lrstr.Rstring = Rstring(s)
	lrstr.count = utf8.RuneCountInString(s)
	return
}

func (s LRstring) Count() int {
	return s.count
}

func (s *LRstring) Concat(s1, s2 LRstring) LRstring {
	s.Rstring = s1.Rstring + s2.Rstring
	s.count = s1.Count() + s2.Count()
	return *s
}
