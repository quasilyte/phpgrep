# phpgrep

[![Go Report Card](https://goreportcard.com/badge/github.com/quasilyte/phpgrep)](https://goreportcard.com/report/github.com/quasilyte/phpgrep)
[![GoDoc](https://godoc.org/github.com/quasilyte/phpgrep?status.svg)](https://godoc.org/github.com/quasilyte/phpgrep)
![Build Status](https://github.com/quasilyte/phpgrep/workflows/Go/badge.svg)

Syntax-aware grep for PHP code.

This repository is used for the library and command-line tool development.
A good source for additional utilities and ready-to-run recipes is [phpgrep-contrib](https://github.com/quasilyte/phpgrep-contrib) repository.

## Overview

`phpgrep` is both a library and a command-line tool.

Library can be used to perform syntax-aware PHP code matching inside Go programs
while binary utility can be used from your favorite text editor or terminal emulator.

It's very close to [structural search and replace](https://www.jetbrains.com/help/phpstorm/structural-search-and-replace.html)
in PhpStorm, but better suited for standalone usage.

In many ways, it's inspired by [github.com/mvdan/gogrep/](https://github.com/mvdan/gogrep/).

See also: ["phpgrep: syntax aware code search"](https://speakerdeck.com/quasilyte/phpgrep-syntax-aware-code-search).

## Quick start

> If you're using VS Code, you might be interested in [vscode-phpgrep](https://marketplace.visualstudio.com/items?itemName=quasilyte.phpgrep) extension.

Download a `phpgrep` binary from the [latest release](https://github.com/quasilyte/phpgrep/releases/), put it somewhere under your `$PATH`.

Run a `-help` command to verify that everything is okay.

```bash
$ phpgrep -help
Usage: phpgrep [flags...] targets pattern [filters...]
Where:
  flags are command-line flags that are listed in -help (see below)
  targets is a comma-separated list of file or directory names to search in
  pattern is a string that describes what is being matched
  filters are optional arguments bound to the pattern

Examples:
  # Find f calls with a single varible argument.
  phpgrep file.php 'f(${"var"})'

  # Like the previous example, but searches inside entire
  # directory recursively and variable names are restricted
  # to $id, $uid and $gid.
  # Also uses -v flag that makes phpgrep output more info.
  phpgrep -v ~/code/php 'f(${"x:var"})' 'x=id,uid,gid'

  # Run phpgrep on 2 folders (recursively).
  phpgrep dir1,dir2 '"some string"'

  # Print only matches, without locations.
  phpgrep -format '{{.Match}}' file.php 'pattern'

  # Print only assignments right-hand side.
  phpgrep -format '{{.rhs}}' file.php '$_ = $rhs'

  # Ignore vendored source code inside project.
  phpgrep --exclude '/vendor/' project/ 'pattern'

Custom output formatting is possible via the -format flag template.
  {{.Filename}} match containing file name
  {{.Line}}     line number where the match started
  {{.Match}}    an entire match string
  {{.x}}        $x submatch string (can be any submatch name)

Exit status:
  0 if something is matched
  1 if nothing is matched
  2 if error occurred

# ... rest of output
```

Create a test file `hello.php`:

```php
<?php
function f(...$xs) {}
f(10);
f(20);
f(30);
f($x);
f();
```

Run `phpgrep` over that file:

```bash
$ phpgrep hello.php 'f(${"x:int"})' 'x!=20'
hello.php:3: f(10)
hello.php:5: f(30)
```

We found all `f` calls with a **single** argument `x` that is `int` literal **not equal** to 20.

Next thing to learn is `${"*"}` matcher.

Suppose you need to match all `foo` function calls that have `null` argument.<br>
`foo` is variadic, so it's unknown where that argument can be located.

This pattern will match `null` arguments at any position: `foo(${"*"}, null, ${"*"})`.

Read [pattern language docs](/pattern_language.md) to learn more about how to write search patterns.

## Recipes

This section contains ready-to-use `phpgrep` patterns.

`srcdir` is a target source directory (can also be a single filename).

### Useful recipes

```bash
# Find arrays with at least 1 duplicated key.
$ phpgrep srcdir '[${"*"}, $k => $_, ${"*"}, $k => $_, ${"*"}]'

# Find where `$x ?: $y` can be applied.
$ phpgrep srcdir '$x ? $x : $y' # Use `$x ?: $y` instead

# Find where `$x ?? $y` can be applied.
$ phpgrep srcdir 'isset($x) ? $x : $y'

# Find in_array calls that can be replaced with $x == $y.
$ phpgrep srcdir 'in_array($x, [$y])'

# Find potential operator precedence issues.
$ phpgrep srcdir '$x & $mask == $y' # Should be ($x & $mask) == $y
$ phpgrep srcdir '$x & $mask != $y' # Should be ($x & $mask) != $y

# Find calls where func args are misplaced.
$ phpgrep srcdir 'stripos(${"str"}, $_)'
$ phpgrep srcdir 'explode($_, ${"str"}, ${"*"})'

# Find new calls without parentheses.
$ phpgrep srcdir 'new $t'

# Find all if statements with a body without {}.
$ phpgrep srcdir 'if ($cond) $x' 'x!~^\{'
# Or without regexp.
$ phpgrep srcdir 'if ($code) ${"expr"}'

# Find all error-supress operator usages.
$ phpgrep srcdir '@$_'

# Find all == (non-strict) comparisons with null.
$ phpgrep srcdir '$_ == null'
```

### Miscellaneous recipes

```bash
# Find all function calls that have at least one var-argument that has _id suffix.
$ phpgrep srcdir '$f(${"*"}, ${"x:var"}, ${"*"})' 'x~.*_id$'

# Find foo calls where the second argument is integer literal.
$ phpgrep srcdir 'foo($_, ${"int"})'
```

### Install from sources

You'll need Go tools to install `phpgrep` from sources.

To install `phpgrep` binary under your `$(go env GOPATH)/bin`:

```bash
go get -v github.com/quasilyte/phpgrep/cmd/phpgrep
```

If `$GOPATH/bin` is under your system `$PATH`, `phpgrep` command should be available after that.
