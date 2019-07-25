package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/quasilyte/phpgrep"
)

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isSpace(c byte) bool {
	return c == ' '
}

func compileFilter(s string) (phpgrep.Filter, error) {
	// TODO(quasilyte): refactor this function.

	var f phpgrep.Filter

	if len(s) == 0 {
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
		values := strings.Split(argument, "|")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		return phpgrep.ValueInListFilter(name, values), nil

	case "!=":
		values := strings.Split(argument, "|")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		return phpgrep.ValueNotInListFilter(name, values), nil

	case "~":
		re, err := regexp.Compile(argument)
		if err != nil {
			return f, fmt.Errorf("argument: %v", err)
		}
		return phpgrep.RegexpFilter(name, re), nil

	case "!~":
		re, err := regexp.Compile(argument)
		if err != nil {
			return f, fmt.Errorf("argument: %v", err)
		}
		return phpgrep.RegexpNotFilter(name, re), nil

	default:
		panic("unreachable")
	}
}
