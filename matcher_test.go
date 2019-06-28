package phpgrep

import (
	"testing"
)

// Match() => bool
// Replace
// Find => matches

type matcherTest struct {
	pattern string
	input   string
}

func runMatchTest(t *testing.T, want bool, tests []*matcherTest) {
	var c Compiler
	for _, test := range tests {
		matcher, err := c.Compile([]byte(test.pattern))
		if err != nil {
			t.Errorf("error compiling %q pattern: %v", test.pattern, err)
			continue
		}

		have := matcher.Match([]byte(test.input))
		if have != want {
			t.Errorf("match results mismatch:\npattern: %q\ninput: %q\nhave: %v\nwant: %v",
				test.pattern, test.input, have, want)
		}
	}
}

func TestMatch(t *testing.T) {
	runMatchTest(t, true, []*matcherTest{
		{"$x", "10"},
	})
}
