This is a port of Neil Frasers [diff_match_patch][dmp] to Go, based on the
Java implementation.

It is a work in progress. At the moment, most of the *diff*
part is available. All corresponding tests have been ported too.

The implementation is based on processing items of type
`string`.  An utility package ./rstring, which is partly
based on Go standard package `exp/utf8string`, helps navigating
runewise (rather than bytewise).  Perhaps it would be preferable
to use `[]rune` instead of `string`, but this hasn't been
tried yet.

The performance on i386 system lies between that of the Java
and the Qt based C++ implementations.
On amd64 systems the Go port is as fast as the Java
implementation (the 6g compiler is able to generate faster code
than 8g, because of the larger number of available registers on 64bit
processors).

[dmp]:	http://code.google.com/p/google-diff-match-patch/
