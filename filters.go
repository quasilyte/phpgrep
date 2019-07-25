package phpgrep

import (
	"bytes"
	"regexp"
)

type filterFunc func([]byte) bool

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
	return func(buf []byte) bool {
		return re.Match(buf)
	}
}

func makeRegexpNotFilter(re *regexp.Regexp) filterFunc {
	return func(buf []byte) bool {
		return !re.Match(buf)
	}
}
