// Diff Match and Patch – tests
// 	Original work: Copyright 2006 Google Inc.
// 	Go port:	Copyright 2012 M. Teichgräber
//
// Use of this source code is governed by the Apache License,
// Version 2.0, that can be found in the LICENSE file.

package dmp

import (
	. "github.com/knieriem/dmp/rstring"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCommonPrefix(t *testing.T) {
	f := func(s1, s2 string) int {
		return len(commonPrefix(s1, s2))
	}
	assertEquals("Null case", 0, f("abc", "xyz"), t)
	assertEquals("Non-Null case", 4, f("1234abcdef", "1234xyz"), t)
	assertEquals("Non-Null case non-ascii", 4, f("1234öabcdef", "1234äxyz"), t)
	assertEquals("Whole case", 4, f("1234", "1234xyz"), t)
}

func TestCommonSuffix(t *testing.T) {
	f := func(s1, s2 string) int {
		return len(commonSuffix(s1, s2))
	}
	assertEquals("Null case", 0, f("abc", "xyz"), t)
	assertEquals("Non-Null case", 4, f("abcdef1234", "xyz1234"), t)
	assertEquals("Non-Null case non-ascii", 4, f("abcdefä1234", "xyzΤ1234"), t)
	assertEquals("Whole case", 4, f("1234", "xyz1234"), t)
}

func TestCommonOverlap(t *testing.T) {
	f := func(s1, s2 string) int {
		return len(commonOverlap(s1, s2))
	}
	// Detect any suffix/prefix overlap
	assertEquals("Null case", 0, f("", "abcd"), t)
	assertEquals("Whole case", 3, f("abc", "abcd"), t)
	assertEquals("No overlap", 0, f("123456", "abcd"), t)
	assertEquals("Overlap", 3, f("123456xxx", "xxxabcd"), t)
	assertEquals("Overlap non-ascii", 5, f("123456äxxx", "äxxxabcd"), t)
}

type halfMatchTest struct {
	name         string
	text1, text2 string
	want         *halfMatch
}

var halfMatchTests = []halfMatchTest{
	{"No match #1", "1234567890", "abcdef", nil},
	{"No match #2", "12345", "23", nil},
	{"Single Match #1", "1234567890", "a345678z", &halfMatch{"12", "90", "a", "z", LRString("345678")}},
	{"Single Match #2", "a345678z", "1234567890", &halfMatch{"a", "z", "12", "90", LRString("345678")}},
	{"Single Match #3", "abc56789z", "1234567890", &halfMatch{"abc", "z", "1234", "0", LRString("56789")}},
	{"Single Match #4", "a23456xyz", "1234567890", &halfMatch{"a", "xyz", "1", "7890", LRString("23456")}},
	{"Single Match #5", "aä23456äöxyz", "123456ä7890", &halfMatch{"aä", "öxyz", "1", "7890", LRString("23456ä")}},
	{"Multiple Matches #1", "121231234123451234123121", "a1234123451234z", &halfMatch{"12123", "123121", "a", "z", LRString("1234123451234")}},
	{"Multiple Matches #2", "x-=-=-=-=-=-=-=-=-=-=-=-=", "xx-=-=-=-=-=-=-=", &halfMatch{"", "-=-=-=-=-=", "x", "", LRString("x-=-=-=-=-=-=-=")}},
	{"Multiple Matches #3", "-=-=-=-=-=-=-=-=-=-=-=-=y", "-=-=-=-=-=-=-=yy", &halfMatch{"-=-=-=-=-=", "", "", "y", LRString("-=-=-=-=-=-=-=y")}},
	// Optimal diff would be -q+x=H-i+e=lloHe+Hu=llo-Hew+y not -qHillo+x=HelloHe-w+Hulloy
	{"Non-optimal halfmatch", "qHilloHelloHew", "xHelloHeHulloy", &halfMatch{"qHillo", "w", "x", "Hulloy", LRString("HelloHe")}},
}

func TestDiffHalfMatch(t *testing.T) {
	// Detect a halfmatch.
	runDiffHalfMatchList(halfMatchTests, false, t)

	runDiffHalfMatchList([]halfMatchTest{
		{"Optimal no halfmatch", "qHilloHelloHew", "xHelloHeHulloy", nil},
	}, true, t)
}

func runDiffHalfMatchList(tests []halfMatchTest, unlimitedTime bool, t *testing.T) {
	for _, test := range tests {
		hm := findHalfMatch(LRString(test.text1), LRString(test.text2), unlimitedTime)
		switch {
		case hm == nil && test.want == nil:
		case hm != nil && test.want != nil && *hm == *test.want:
		default:
			t.Error(test.name, hm, test.want)
		}
	}
}

func TestDiffLinesToChars(t *testing.T) {
	f := diffLinesToChars

	d := &linesDesc{
		"\u0001\u0002\u0001",
		"\u0002\u0001\u0002",
		[]string{"", "alpha\n", "beta\n"},
	}
	assertEquals("Shared lines", d, f("alpha\nbeta\nalpha\n", "beta\nalpha\nbeta\n"), t)

	d = &linesDesc{
		"",
		"\u0001\u0002\u0003\u0003",
		[]string{"", "alpha\r\n", "beta\r\n", "\r\n"},
	}
	assertEquals("Empty string and blank lines", d, f("", "alpha\r\nbeta\r\n\r\n\r\n"), t)

	d = &linesDesc{
		"\u0001",
		"\u0002",
		[]string{"", "a", "b"},
	}
	assertEquals("No linebreaks", d, f("a", "b"), t)

	text, d := build300LinesTest(t)
	assertEquals("More than 256", d, f(text, ""), t)
}

func TestDiffCharsToLines(t *testing.T) {
	// First check that Diff equality works.
	assertTrue("Equality #1", Diff{Equal, "a"} == Diff{Equal, "a"}, t)
	assertEquals("Equality #2", Diffs{{Equal, "a"}}, Diffs{{Equal, "a"}}, t)

	// Convert chars up to lines.
	diffs := diffList("=<\u0001\u0002\u0001> +<\u0002\u0001\u0002>")
	tmpVector := []string{"", "alpha\n", "beta\n"}
	diffCharsToLines(diffs, tmpVector)
	assertEquals("Shared lines", diffList("=<alpha\nbeta\nalpha\n> +<beta\nalpha\nbeta\n>"), diffs, t)

	lines, d := build300LinesTest(t)
	diffs = Diffs{{Delete, d.chars1}}
	diffCharsToLines(diffs, d.lines)
	assertEquals("More than 256.", Diffs{{Delete, lines}}, diffs, t)
}

func build300LinesTest(t *testing.T) (text string, d *linesDesc) {
	// More than 256 to reveal any 8-bit limitations.
	n := 300
	text = ""
	d = new(linesDesc)
	d.lines = []string{""}
	d.chars1 = ""
	d.chars2 = ""
	for x := 1; x < n+1; x++ {
		s := strconv.Itoa(x) + "\n"
		d.lines = append(d.lines, s)
		text += s
		d.chars1 += string(rune(x))
	}
	assertEquals("d.lines", n+1, len(d.lines), t)
	assertEquals("d.chars1", n, Rstring(d.chars1).Count(), t)
	return
}

func TestDiffCleanupMerge(t *testing.T) {
	f := func(desc string) Diffs {
		diffs := diffList(desc)
		return diffs.CleanupMerge()
	}
	assertEquals("Null case", Diffs{}, f(""), t)
	for _, x := range []struct{ name, input, result string }{

		{"No change case", "=<a> -<b> +<c>", "=<a> -<b> +<c>"},
		{"Merge equalities", "=<a> =<b> =<c>", "=<abc>"},
		{"Merge deletions", "-<a> -<b> -<c>", "-<abc>"},
		{"Merge insertions", "+<a> +<b> +<c>", "+<abc>"},
		{"Merge interweave", "-<a> +<b> -<c> +<d> =<e> =<f>", "-<ac> +<bd> =<ef>"},
		{"Prefix and suffix detection", "-<a> +<abc> -<dc>", "=<a> -<d> +<b> =<c>"},
		{
			"Prefix and suffix detection with equalities",
			"=<x> -<a> +<abc> -<dc> =<y>",
			"=<xa> -<d> +<b> =<cy>",
		},
		{"Slide edit left", "=<a> +<ba> =<c>", "+<ab> =<ac>"},
		{"Slide edit right", "=<c> +<ab> =<a>", "=<ca> +<ba>"},
		{"Slide edit left recursive", "=<a> -<b> =<c> -<ac> =<x>", "-<abc> =<acx>"},
		{"Slide edit right recursive", "=<x> -<ca> =<c> -<b> =<a>", "=<xca> -<cba>"},
	} {
		assertEquals(x.name, diffList(x.result), f(x.input), t)
	}
}

func TestDiffCleanupSemanticLossless(t *testing.T) {
	f := func(desc string) Diffs {
		diffs := diffList(desc)
		return diffs.CleanupSemanticLossless()
	}
	for _, x := range []struct{ name, input, result string }{
		{"Null case", "", ""},
		{
			"Blank lines.",
			"=<AAA\r\n\r\nBBB> +<\r\nDDD\r\n\r\nBBB> =<\r\nEEE>",
			"=<AAA\r\n\r\n> +<BBB\r\nDDD\r\n\r\n> =<BBB\r\nEEE>",
		}, {
			"Line boundaries.",
			"=<AAA\r\nBBB> +< DDD\r\nBBB> =< EEE>",
			"=<AAA\r\n> +<BBB DDD\r\n> =<BBB EEE>",
		}, {
			"Word boundaries.",
			"=<The c> +<ow and the c> =<at.>",
			"=<The > +<cow and the > =<cat.>",
		}, {
			"Alphanumeric boundaries.",
			"=<The-c> +<ow-and-the-c> =<at.>",
			"=<The-> +<cow-and-the-> =<cat.>",
		},
		{"Hitting the start", "=<a> -<a> =<ax>", "-<a> =<aax>"},
		{"Hitting the end", "=<xa> -<a> =<a>", "=<xaa> -<a>"},
		{
			"Sentence boundaries",
			"=<The xxx. The > +<zzz. The > =<yyy.>",
			"=<The xxx.> +< The zzz.> =< The yyy.>",
		},
	} {
		assertEquals(x.name, diffList(x.result), f(x.input), t)
	}
}

// | awk '/diffs =/{ sub("^ *diffs = *", ""); sub(";$", ""); d = $0; next}
// /^  *"/{sub("diffs\\);$", d); sub("^ *", ""); print "\t{ " $0 " },"}
// '
func TestDiffCleanupSemantic(t *testing.T) {
	f := func(desc string) Diffs {
		diffs := diffList(desc)
		return diffs.CleanupSemantic()
	}
	for _, x := range []struct{ name, input, result string }{
		{"Null case", "", ""},
		{
			"No elimination #1.",
			"-<ab> +<cd> =<12> -<e>",
			"-<ab> +<cd> =<12> -<e>",
		}, {
			"No elimination #2.",
			"-<abc> +<ABC> =<1234> -<wxyz>",
			"-<abc> +<ABC> =<1234> -<wxyz>",
		},
		{"Simple elimination.", "-<a> =<b> -<c>", "-<abc> +<b>"},
		{
			"Backpass elimination.",
			"-<ab> =<cd> -<e> =<f> +<g>",
			"-<abcdef> +<cdfg>",
		}, {
			"Double backpass elimination.",
			"-<123> =<xyz> -<ab> =<cd> -<e> =<f> +<g>",
			"-<123xyzabcdef> +<xyzcdfg>",
		}, {
			"Multiple elimination.",
			"+<1> =<A> -<B> +<2> =<_> +<1> =<A> -<B> +<2>",
			"-<AB_AB> +<1A2_1A2>",
		}, {
			"Word boundaries.",
			"=<The c> -<ow and the c> =<at.>",
			"=<The > -<cow and the > =<cat.>",
		},
		{"No overlap elimination.", "-<abcxx> +<xxdef>", "-<abcxx> +<xxdef>"},
		{"Overlap elimination.", "-<abcxxx> +<xxxdef>", "-<abc> =<xxx> +<def>"},
		{
			"Reverse overlap elimination.",
			"-<xxxabc> +<defxxx>",
			"+<def> =<xxx> -<abc>",
		}, {
			"Two overlap eliminations.",
			"-<abcd1212> +<1212efghi> =<----> -<A3> +<3BC>",
			"-<abcd> =<1212> +<efghi> =<----> -<A> =<3> +<BC>",
		},
	} {
		assertEquals(x.name, diffList(x.result), f(x.input), t)
	}
}

func TestDiffCleanupEfficiency(t *testing.T) {
	// Cleanup operationally trivial equalities

	for _, x := range []struct {
		name          string
		editCost      int
		input, result string
	}{
		{"Null case", 4, "", ""},
		{
			"No elimination", 4,
			"-<ab> +<12> =<wxyz> -<cd> +<34>",
			"-<ab> +<12> =<wxyz> -<cd> +<34>",
		}, {
			"Four-edit elimination", 4,
			"-<ab> +<12> =<xyz> -<cd> +<34>",
			"-<abxyzcd> +<12xyz34>",
		}, {
			"Three-edit elimination", 4,
			"+<12> =<x> -<cd> +<34>",
			"-<xcd> +<12x34>",
		}, {
			"Backpass elimination", 4,
			"-<ab> +<12> =<xy> +<34> =<z> -<cd> +<56>",
			"-<abxyzcd> +<12xy34z56>",
		}, {
			"Double backpass elimination", 4,
			"+<ab> -<cd> =<12> -<ef> =<3> -<gh> =<4> -<xy> +<zz>",
			"-<cd12ef3gh4xy> +<ab1234zz>",
		}, {
			"Safe backpass elimination", 4,
			"+<a> -<b> =<1> +<c> =<22> -<d> =<3> -<e> +<f>",
			"-<b122d3e> +<a1c223f>",
		}, {
			"High cost elimination", 5,
			"-<ab> +<12> =<wxyz> -<cd> +<34>",
			"-<abwxyzcd> +<12wxyz34>",
		},
	} {
		diffs := diffList(x.input)
		assertEquals(x.name, diffList(x.result), diffs.CleanupEfficiency(x.editCost), t)
	}
}

func TestDiffPrettyHTML(t *testing.T) {
	diffs := Diffs{{Equal, "a\n"}, {Delete, "<B>b</B>"}, {Insert, "c&d"}}
	assertEquals("-", "<span>a&para;<br></span><del style=\"background:#ffe6e6;\">&lt;B&gt;b&lt;/B&gt;</del><ins style=\"background:#e6ffe6;\">c&amp;d</ins>", diffs.PrettyHTML(), t)
}

func TestDiffText(t *testing.T) {
	// Compute the source and destination texts.
	diffs := diffList("=<jump> -<s> +<ed> =< over > -<the> +<a> =< lazy>")
	assertEquals("diff_text1", "jumps over the lazy", diffs.Text1(), t)
	assertEquals("diff_text2", "jumped over a lazy", diffs.Text2(), t)
}

func TestDiffLevenshtein(t *testing.T) {
	diffs := diffList("-<abc> +<1234> =<xyz>")
	assertEquals("Levenshtein with trailing equality", 4, diffs.Levenshtein(), t)

	diffs = diffList("=<xyz> -<abc> +<1234>")
	assertEquals("Levenshtein with leading equality", 4, diffs.Levenshtein(), t)

	diffs = diffList("-<abc> =<xyz> +<1234>")
	assertEquals("Levenshtein with middle equality", 7, diffs.Levenshtein(), t)
}

func TestDiffBisect(t *testing.T) {
	// Normal.
	a := NewIRstring("cat")
	b := NewIRstring("map")
	// Since the resulting diff hasn't been normalized, it would be ok if
	// the insertion and deletion pairs are swapped.
	// If the order changes, tweak this test as required.
	diffs := diffList("-<c> +<m> =<a> -<t> +<p>")
	d := new(differ)
	d.bisect(a, b)
	assertEquals("Normal", diffs, d.Diffs, t)

	// Timeout.
	diffs = diffList("-<cat> +<map>")
	d = new(differ)
	// fake timeout
	d.deadLine = time.Now().Add(-time.Second)
	d.bisect(a, b)
	assertEquals("Timeout", diffs, d.Diffs, t)
}

func TestDiffMain(t *testing.T) {
	// Perform a trivial diff.
	runDiffTests(t, time.Second, []diffTest{
		// Perform a trivial diff.
		{"Null case", "", "", ""},
		{"Equality", "abc", "abc", "=<abc>"},
		{"Simple insertion", "abc", "ab123c", "=<ab> +<123> =<c>"},
		{"Simple deletion", "a123bc", "abc", "=<a> -<123> =<bc>"},
		{"Two insertions", "abc", "a123b456c", "=<a> +<123> =<b> +<456> =<c>"},
		{"Two deletions", "a123b456c", "abc", "=<a> -<123> =<b> -<456> =<c>"},
	})

	// Perform a real diff.
	// Switch off the timeout.
	runDiffTests(t, 0, []diffTest{
		{"Simple case #1.", "a", "b", "-<a> +<b>"},
		{
			"Simple case #2.",
			"Apples are ä fruit.",
			"Bananas are älso fruit.",
			"-<Apple> +<Banana> =<s are ä> +<lso> =< fruit.>",
		},
		{"Simple case #3.", "ax\t", "\u0680x\000", "-<a> +<\u0680> =<x> -<\t> +<\000>"},
		{"Overlap #1.", "1ayb2", "abxab", "-<1> =<a> -<y> =<b> -<2> +<xab>"},
		{"Overlap #2.", "abcy", "xaxcxabc", "+<xaxcx> =<abc> -<y>"},
		{
			"Overlap #3.",
			"ABCDa=bcd=efghijklmnopqrsEFGHIJKLMNOefg",
			"a-bcd-efghijklmnopqrs",
			"-<ABCD> =<a> -<=> +<-> =<bcd> -<=> +<-> =<efghijklmnopqrs> -<EFGHIJKLMNOefg>",
		}, {
			"Large equality.",
			"a [[Pennsylvania]] and [[New",
			" and [[Pennsylvania]]",
			"+< > =<a> +<nd> =< [[Pennsylvania]]> -< and [[New>",
		},
	})

	a := "`" + `Twas brillig, and the slithy toves
Did gyre and gimble in the wabe:
All mimsy were the borogoves,
And the mome raths outgrabe.
`
	b := `I am the very model of a modern major general,
I've information vegetable, animal, and mineral,
I know the kings of England, and I quote the fights historical,
From Marathon to Waterloo, in order categorical.
`
	// Increase the text lengths by 1024 times to ensure a timeout.
	for x := 0; x < 10; x++ {
		a += a
		b += b
	}
	timeout := time.Duration(100) * time.Millisecond
	t0 := time.Now()
	DiffMain(a, b, false, timeout)
	Δt := time.Now().Sub(t0)
	t.Log("Δt =", Δt)

	// Test that we took at least the timeout period.
	assertTrue("Timeout min", timeout <= Δt, t)

	// Test that we didn't take forever (be forgiving).
	// Theoretically this test could fail very occasionally if the
	// OS task swaps or locks up for a second at the wrong moment.
	assertTrue("Timeout max", timeout*2 > Δt, t)

	// Test the linemode speedup.
	// Must be long to pass the 100 char cutoff.
	aDig := `1234567890
1234567890
1234567890
1234567890
1234567890
1234567890
1234567890
1234567890
1234567890
1234567890
1234567890
1234567890
1234567890
`
	b = `abcdefghij
abcdefghij
abcdefghij
abcdefghij
abcdefghij
abcdefghij
abcdefghij
abcdefghij
abcdefghij
abcdefghij
abcdefghij
abcdefghij
abcdefghij
`
	assertEquals("Simple line-mode.", DiffMain(aDig, b, true, 0), DiffMain(aDig, b, false, 0), t)

	a = `1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890`
	b = `abcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghij`
	assertEquals("Single line-mode.", DiffMain(a, b, true, 0), DiffMain(a, b, false, 0), t)

	//	b = "abcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n";
	//	String[] texts_linemode = diff_rebuildtexts(dmp.diff_main(aDig, b, true));
	//	String[] texts_textmode = diff_rebuildtexts(dmp.diff_main(aDig, b, false));
	//	assertArrayEquals("diff_main: Overlap line-mode.", texts_textmode, texts_linemode);
}

type diffTest struct {
	name, text1, text2, result string
}

func runDiffTests(t *testing.T, timeout time.Duration, tests []diffTest) {
	for i := range tests {
		test := &tests[i]
		result := DiffMain(test.text1, test.text2, false, timeout)
		assertEquals(test.name, diffList(test.result), result, t)
	}
}

func diffList(desc string) (diffs Diffs) {
	dl := strings.Split(desc, ">")
	if desc == "" {
		return
	}
	diffs = make([]Diff, len(dl)-1)
	for i, diff := range dl {
		if diff == "" {
			continue
		}
		if diff[0] == ' ' {
			diff = diff[1:]
		}
		op := 0
		switch diff[0] {
		case '=':
			op = Equal
		case '+':
			op = Insert
		case '-':
			op = Delete
		}
		diffs[i] = Diff{op, diff[2:]}
	}
	return
}

func assertTrue(descr string, ok bool, t *testing.T) {
	if !ok {
		t.Error(descr)
	}
}

func assertEquals(descr string, want, have interface{}, t *testing.T) {
	ok := false
	switch want := want.(type) {
	case int:
		ok = want == have.(int)

	case string:
		ok = want == have.(string)

	case Diffs:
		have := have.(Diffs)
		if len(want) == len(have) {
			for i := range want {
				if have[i] != want[i] {
					goto sorry
				}
			}
			ok = true
		sorry:
		}

	case *linesDesc:
		have := have.(*linesDesc)
	s:
		switch {
		case have.chars1 != want.chars1:
		case have.chars2 != want.chars2:
		case len(have.lines) != len(have.lines):
		default:
			for i := range have.lines {
				if have.lines[i] != want.lines[i] {
					break s
				}
			}
			ok = true
		}
	}
	if !ok {
		t.Errorf("%s.\n\twant: %v\n\thave: %v\n", descr, want, have)
	}
}
