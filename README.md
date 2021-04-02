This is a port of the _diff_ part of Neil Fraser's [diff_match_patch][dmp] to Go,
derived from the Java implementation.
All diff-related tests have been ported too.

The implementation is based on processing items of type
`string`.  An utility package ./rstring, which is partly
based on Go standard package `exp/utf8string`, helps navigating
runewise (rather than bytewise).  Perhaps it would be preferable
to use `[]rune` instead of `string`, but this hasn't been
tried yet.

Measured in 2012,
the performance on a i386 system was between that of the Java
and the Qt based C++ implementations.
On amd64 systems the Go port performed as fast as the Java
implementation (the 6g compiler is able to generate faster code
than 8g, because of the larger number of available registers on 64bit
processors).

[dmp]:	https://github.com/google/diff-match-patch
