# phpgrep

[![Go Report Card](https://goreportcard.com/badge/github.com/quasilyte/phpgrep)](https://goreportcard.com/report/github.com/quasilyte/phpgrep)
[![GoDoc](https://godoc.org/github.com/quasilyte/phpgrep?status.svg)](https://godoc.org/github.com/quasilyte/phpgrep)
[![Build Status](https://travis-ci.org/quasilyte/phpgrep.svg?branch=master)](https://travis-ci.org/quasilyte/phpgrep)

Syntax-aware grep for PHP code search.

It's very close to [structural search and replace](https://www.jetbrains.com/help/phpstorm/structural-search-and-replace.html)
in PhpStorm, but better suited for standalone usage.

In many ways, inspired by [github.com/mvdan/gogrep/](https://github.com/mvdan/gogrep/).

## Overview

Both a library and a command-line tool.

Library can be used to perform syntax-aware PHP code matching inside Go programs
while binary utility can be used from your favorite text editor or terminal emulator.

## Useful recipes

```bash
# Find arrays with at least 1 duplicated key.
$ phpgrep srcdir '[${"*"}, $k => $_, ${"*"}, $k => $_, ${"*"}]'

# Find sloppy strcmp uses.
$ phpgrep srcdir 'strcmp($s1, $s2) > 0'   # Use `$s1 > $s2`
$ phpgrep srcdir 'strcmp($s1, $s2) < 0'   # Use `$s1 < $s2`
$ phpgrep srcdir 'strcmp($s1, $s2) === 0' # Use `$s1 === $s2`

# Find new calls without parentheses.
$ phpgrep srcdir 'new $t'
```
