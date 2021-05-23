# phpgrep user manual

## Introduction

`phpgrep` is a tool that can be used to perform a structural search of the PHP code.

It can replace a `grep` for you.

It works on an AST level instead of the text level. That means that the search patterns encode the AST structure of the potential match.

It's very close to the [structural search and replace](https://www.jetbrains.com/help/phpstorm/structural-search-and-replace.html)
in PhpStorm, but better suited for standalone usage.

This document explains some core concepts and covers the `phpgrep` CLI API.

## Writing patterns

| topic | grep | phpgrep |
|---|---|---|
| Search patterns | Regular expressions | AST patterns |
| Operates on | Text level | Strucural level |
| Minimal search unit | Character | AST element |

As `phpgrep` uses AST elements for its patterns, they should be syntactically correct for the PHP parser (with some very rare exceptions).

`f(` is valid from the text point of view, but it's not valid from the PHP grammar point of view; it should be `f()` to become a valid search pattern.

You can't search for the incomplete AST element, but you can search for the arbitrary part of the AST trees.

`phpgrep` patterns consist of two main parts:

* Literal matching, where the text matches the exact code fragments as described in a pattern
* Matcher expressions that can match arbitrary AST nodes

Literal matching is very easy to understand.

| Pattern | Matches | Doesn't match |
|---|---|---|
| `f()` | `f()`, `f ()` | `g()`, `100` |
| `[1, 2]` | `[1, 2]` | `g()`, `100`, `[1]` |
| `var_dump("hello")` | `var_dump("hello")` | `var_dump("hi")` |

To make patterns much more useful, you can use matcher expressions to make them more dynamic.

| Pattern | Matches | Doesn't match |
|---|---|---|
| `f($x)` | `f(1)`, `f(["ok"])` | `f()`, `g(1)` |
| `[$x, $x]` | `[1, 1]`, `["a", "a"]` | `[1, 2]` |
| `[$_, $_]` | `[1, 1]`, `[1, 2]` | `[1, 2, 3]` |
| `foo(${"*"}, true)` | `foo(true)`, `foo($v, true)` | `f()`, `f(false)` |
| `$_->build()` | `$x->build()`, `$list[$i]->build()` | `$x->run()`, `build()` |

To keep it simple:

* `$` variables are used to fill the dynamic part of the pattern
* Non-variables match "as is"
* Repeated variable names require all submatches to be identical
* `$_` is a special variable that don't require `_` submatches to be identical
* `${"*"}` is like `.*` in regular expressions; use it for the optional or variadic parts

Every variable (matcher expression) forms a **named** submatch that can be referenced in filters and `--format` pattern string (read below).

Read the [pattern language overview](/pattern_language.md) to learn more about the matcher expressions.

## Filters

After pattern is matched, additional filters can be applied to either accept or reject the match.

Filters can only be applied to **named** submatches.

```
filter = <name> operator argument
operator = <see list of supported ops below>
argument = <depends on the operator>
```

Filters are connected like a pipeline.
If the first filter failed, a second filter will not be executed and the match will be rejected.

This is an impossible filter list:

```
'x=1' 'x=2'
```

`x` is required to be equal to `1` and then it compared to `2`.

**or**-like behavior can be encoded in several operator arguments using `,`:

```
'x=1,2'
```

### `~` filter

The `~` filter matches matched node source text against regular expression.

> Note: it uses **original** source text, not the printed AST node representation.
> It means that you need to take code formatting into account.

### `!~` filter

Opposite of `~`. Matches only when given regexp is not matched.

### `=` filter

| Class | Effect | Example |
|---|---|---|
| `*` | Sub-pattern | `x=[$_]` |
| `+` |  Sub-pattern | `x=${"var"}` |
| `int` | Literal matching | `x=1,20,300` |
| `float` | Literal matching | `x=5.6` |
| `num` | Literal matching | `x=1,1.5` |
| `str` | Literal matching | `x="foo","bar"` |
| `char` | Literal matching | `x="x","y"` |
| `const` | Literal matching | `x=FOO,C::BAR` |
| `var` | Literal matching | `x=$length,$len` |
| `expr` | Literal matching | `x=1` |

Sub-pattern can include any valid PPL text.

Literal matching accept a comma-separated lists of permitted values.

> Note: `*` and `+` filters (sub-patterns) are not implemented yet.

For strings you need to use quotes, so there is no problem with having `,` inside them.

### `!=` filter

Opposite of `=`. Matches only when `=` would not match.

## Command line arguments

### `--limit` argument

By default, `phpgrep` stops when it finds 1000 matches.

This is due to the fact that it's easy to write a too generic pattern and get overwhelming results on the big code bases.

To increase this limit, specify the `--limit` argument with some big value you're comfortable with.

If you want to set it to the max value, use `--limit 0`.

### `--format` argument

Sometimes you want to print the result in some specific way.

Suppose that we have this `target.php` file:

```php
function f() {
    die("unimplemented"); // Should never happen
}
```

```bash
$ phpgrep target.php 'die($_)'
target.php:3:     die("unimplemented"); // Should never happen
```

As you see, it prints the whole line and the match location, not just the match. If we want `phpgrep` to output only the matched part, we can use the `--format` flag.

```bash
$ phpgrep --format '{{.Match}}' target.php 'die($_)'
die("unimplemented")
```

The default format value is `{{.Filename}}:{{.Line}}: {{.MatchLine}}`.

Several template variables are available:

```
  {{.Filename}}  match containing file name
  {{.Line}}      line number where the match started
  {{.MatchLine}} a source code line that contains the match
  {{.Match}}     an entire match string
  {{.x}}         $x submatch string (can be any submatch name)
```

Submatches can be useful in your `--format` string if you want to aim your refactoring.

Imagine that we want to replace the `array_push` with `[]=`. Our `target.php` file:

```php
function f() {
    array_push($data[0], $elem);
}
```

```bash
phpgrep --format '{{.arr}}[] = {{.x}}' target.php 'array_push($arr, $x)'
$data[0][] = $elem
```

Now let's assume that we need a stream of JSON objects:

```bash
phpgrep --no-color --format '{"old":"{{.Match}}","new":"{{.arr}}[] = {{.x}}"}' target.php  'array_push($arr, $x)'
{"old":"array_push($data[0], $elem)","new":"$data[0][] = $elem"}
```

We use `--no-color` to avoid ANSI escapes inserted into the `{{.Match}}` variable.

This mechanism can be useful in combination with your other automation tools.

### `--abs` argument

By default, `phpgrep` prints the relative filenames in the output.

Technically speaking, it sets `{{.Filename}}` variable to the relative file name.

To get an absolute path there, use `--abs` argument.

```bash
$ phpgrep target.php 'array_push($_, $_)'
target.php:3:     array_push($data[0], $elem);

$ phpgrep --abs target.php 'array_push($_, $_)'
/home/quasilyte/target.php:3:     array_push($data[0], $elem);
```

### `--strict-syntax` argument

Unless told otherwise, `phpgrep` would try to match similar operations as identical, so `array(1)` and `[1]` are considered to be equal.

If you don't want this behavior to occure, normalization can be disabled with `--strict-syntax` flag.

Some normalization rules are listed below, for convenience.

| Pattern | Matches (if there is no `--strict-syntax`) | Comment |
|---|---|---|
| `array(...)` | `array(...)`, `[...]` | Array alt syntax |
| `[...]` | `array(...)`, `[...]` | Array alt syntax |
| `list(...) =` | `list(...) =`, `[...] =` | List alt syntax |
| `[...] =` | `array(...) =`, `[...] =` | List alt syntax |
| `new T` | `new T()`, `new T` | Optional constructor parens |
| `new T()` | `new T()`, `new T` | Optional constructor parens |
| `0x1` | `0x1`, `1`, `0b1` | Int literals normalization |
| `0.1` | `0.1`, `.1`, `0.10` | Float literals normalization |
| `doubleval($x)` | `doubleval($x)`, `floatval($x)` | Func alias resolving |
| `"str"` | `"str"`, `'str'` | Single and double quotes |
| `f($x, $x)` | `f(1, 1)`, `f((1), 1)`, `f(1, (1))` | Args parens reduction |
| `[$x, $x]` | `[1, 1]`, `[(1), 1]`, `[1, (1)]` | Array item parens reduction |

There is a simple rule on how to decide whether you need fuzzy matching or not:

* If you're looking for the exact syntax, use `--strict-syntax` flag
* If you're looking for some generic code pattern, don't use `--strict-syntax` flag

### `--case-sensitive` argument

`phpgrep` tries to match the PHP behavior when it comes to the case sensitivity.

In PHP these symbols are case insensitive:

* Functions
* Class method names (both static and instance)
* Class names

Given this `target.php`:

```php
function f() {}
f();
F();
```

We can get these results:

```bash
$ phpgrep target.php 'f()'
target.php:3: f();
target.php:4: F();

$ phpgrep target.php 'F()'
target.php:3: f();
target.php:4: F();

$ phpgrep --case-sensitive target.php 'f()'
target.php:3: f();

$ phpgrep --case-sensitive target.php 'F()'
target.php:4: F();
```

### Multi-line mode, `--m` argument

Some patterns may match a code that spans across the multiple lines.

By default, `phpgrep` replaces all newlines with `\n` sequence, so you can treat all matches as strings without newlines.

If you want to avoid that behavior, `--m` argument can be used.

```php
// target.php
var_dump(
    1,
    2
);
```

```bash
$ phpgrep target.php 'var_dump(${"*"})'
target.php:2: var_dump(\n    1,\n    2\n);

$ phpgrep --m target.php 'var_dump(${"*"})'
target.php:2: var_dump(
    1,
    2
);
```

### Output color

`phpgrep` inserts ANSI color escapes by the default.

You can disable this behavior with the `--no-color` flag.

The default color scheme looks like this:

* `dark-red` for `{{.Match}}` and parts of the `{{.MatchLine}}`
* `dark-magenta` for the `{{.Filename}}`
* `dark-green` for the `{{.Line}}`

You can override these colors with these CLI arguments:

| Argument | Format variable | Env variable |
|---|---|---|
| `--color-filename` | `{{.Filename}}` | `$PHPGREP_COLOR_FILENAME` |
| `--color-line` | `{{.Line}}` | `$PHPGREP_COLOR_LINE` |
| `--color-match` | `{{.Match}}` and `{{.MatchLine}}` | `$PHPGREP_COLOR_MATCH` |

To simplify the configuration, `phpgrep` looks up the env variables as well. If both env variable and the command line flags are specified, command line flags win.

List of supported colors:

* white (empty string argument or `white`)
* `dark-red`
* `red`
* `dark-blue`
* `blue`
* `dark-magenta`
* `magenta`
* `dark-green`
* `green`

### `--progress` argument

In order to work faster, `phpgrep` doesn't print any search results until it finds them all (or reaches the `--limit`).

If you're searching through a big (several millions SLOC) project, it could take a few seconds to complete. As it might look like the program hangs, `phpgrep` prints its progress in this manner:

```
N matches so far, processed M files
```

Where `N` and `M` are variables that will change over time.

`phpgrep` has 3 progress reporting modes:

* `none` which is a silent mode, no progress will be printed
* `append` is a simplest mode that just writes one log message after another
* `update` is a more user-friendly mode that will use `\r` to update the message (default)

To override the default mode, use `--progress` argument.

> Note: logs are written to the `stderr` while matches are written to the `stdout`.

### `--exclude` argument

If you want to ignore some directories or files, use `--exclude` argument.

Here is an example that ignores everything under the `vendor/` folder:

```bash
$ phpgrep --exclude 'vendor/' . '<pattern>'
```

`--exclude` accepts a regexp argument.

## Usage examples

Sometimes it's easier to understand things by examples.

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

# Find all function calls that have at least one var-argument that has _id suffix.
$ phpgrep srcdir '$f(${"*"}, ${"x:var"}, ${"*"})' 'x~.*_id$'

# Find foo calls where the second argument is integer literal.
$ phpgrep srcdir 'foo($_, ${"int"})'
```
