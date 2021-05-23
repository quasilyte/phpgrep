# phpgrep user manual

## Introduction

`phpgrep` is a tool that can be used to perform a structural search of the PHP code.

It can replace a `grep` for you.

It works on an AST level instead of the text level. That means that the search patterns encode the AST structure of the potential match.

It's very close to the [structural search and replace](https://www.jetbrains.com/help/phpstorm/structural-search-and-replace.html)
in PhpStorm, but better suited for standalone usage.

In many ways, it's inspired by [github.com/mvdan/gogrep/](https://github.com/mvdan/gogrep/).

If you like slides, check out ["phpgrep - syntax aware code search"](https://speakerdeck.com/quasilyte/phpgrep-syntax-aware-code-search).

This document explains some core concepts and covers the `phpgrep` CLI API.

## `--limit` argument

By default, `phpgrep` stops when it finds 1000 matches.

This is due to the fact that it's easy to write a too generic pattern and get overwhelming results on the big code bases.

To increase this limit, specify the `--limit` argument with some big value you're comfortable with.

If you want to set it to the max value, use `--limit 0`.

## `--format` argument

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

## `--abs` argument

By default, `phpgrep` prints the relative filenames in the output.

Technically speaking, it sets `{{.Filename}}` variable to the relative file name.

To get an absolute path there, use `--abs` argument.

```bash
$ phpgrep target.php 'array_push($_, $_)'
target.php:3:     array_push($data[0], $elem);

$ phpgrep --abs target.php 'array_push($_, $_)'
/home/quasilyte/target.php:3:     array_push($data[0], $elem);
```

## `--case-sensitive` argument

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

## Multi-line mode, `--m` argument

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

## Output color

`phpgrep` inserts ANSI color escapes by the default.

You can disable this behavior with the `--no-color` flag.

The default color scheme looks like this:

* `dark-red` for `{{.Match}}` and parts of the `{{.MatchLine}}`
* `dark-magenta` for the `{{.Filename}}`
* `dark-green` for the `{{.Line}}`

You can override these colors with these CLI arguments:

| Argument | Format variable | Env variable |
|---|---|
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
