package phpgrep

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func compileFilter(s string) (phpgrepFilter, error) {
	// TODO(quasilyte): refactor this function.

	var f phpgrepFilter

	if s == "" {
		return f, fmt.Errorf("filter must include matcher name, operator and its argument")
	}

	pos := 0
	for i := 0; i < len(s); i++ {
		if !isLetter(s[i]) {
			break
		}
		pos++
	}

	if pos == 0 {
		return f, fmt.Errorf("expected matcher name, found %q", s[0])
	}

	name := s[:pos]

	if pos == len(s) {
		return f, fmt.Errorf("missing operator")
	}

	var op string
	switch s[pos] {
	case '=':
		op = "="
	case '~':
		op = "~"
	case '!':
		if pos+1 == len(s) {
			return f, fmt.Errorf(`operator: expected "!=" or "!~", found only "!"`)
		}
		switch s[pos+1] {
		case '=':
			op = "!="
		case '~':
			op = "!~"
		default:
			return f, fmt.Errorf(`operator: expected "!=" or "!~", found "!%c"`, s[pos+1])
		}

	default:
		return f, fmt.Errorf("unexpected operator %q", s[pos])
	}

	pos += len(op)

	argument := s[pos:]

	switch op {
	case "=":
		values := strings.Split(argument, ",")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		return valueInListFilter(name, values), nil

	case "!=":
		values := strings.Split(argument, ",")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		return valueNotInListFilter(name, values), nil

	case "~":
		re, err := regexp.Compile(argument)
		if err != nil {
			return f, fmt.Errorf("argument: %v", err)
		}
		return regexpFilter(name, re), nil

	case "!~":
		re, err := regexp.Compile(argument)
		if err != nil {
			return f, fmt.Errorf("argument: %v", err)
		}
		return regexpNotFilter(name, re), nil

	default:
		panic("unreachable")
	}
}

type phpgrepFilter struct {
	name string
	fn   filterFunc
}

type filterFunc func([]byte) bool

func valueNotInListFilter(name string, values []string) phpgrepFilter {
	return phpgrepFilter{name: name, fn: makeValueNotInListFilter(values)}
}

func valueInListFilter(name string, values []string) phpgrepFilter {
	return phpgrepFilter{name: name, fn: makeValueInListFilter(values)}
}

func regexpNotFilter(name string, re *regexp.Regexp) phpgrepFilter {
	return phpgrepFilter{name: name, fn: makeRegexpNotFilter(re)}
}

func regexpFilter(name string, re *regexp.Regexp) phpgrepFilter {
	return phpgrepFilter{name: name, fn: makeRegexpFilter(re)}
}

func makeValueNotInListFilter(values []string) filterFunc {
	f := makeValueInListFilter(values)
	return func(buf []byte) bool {
		return !f(buf)
	}
}

func makeValueInListFilter(values []string) filterFunc {
	list := make([][]byte, len(values))
	for i := range values {
		list[i] = []byte(values[i])
	}

	return func(buf []byte) bool {
		for _, v := range list {
			if bytes.Equal(buf, v) {
				return true
			}
		}
		return false
	}
}

func makeRegexpFilter(re *regexp.Regexp) filterFunc {
	return re.Match
}

func makeRegexpNotFilter(re *regexp.Regexp) filterFunc {
	return func(buf []byte) bool {
		return !re.Match(buf)
	}
}
