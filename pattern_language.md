# phpgrep pattern language

This file serves as a main documentation source for the pattern language used inside phpgrep.

We'll refer to it as PPL (phpgrep pattern language) for brevity.

## Overview

Syntax-wise, PPL is 100% compatible with PHP.

In fact, it only changes the semantics of some syntax constructions without adding any
new syntax forms to the PHP language. It means that phpgrep patterns can be parsed by
any parser that can handle PHP.

The patterns describe the program parts (syntax trees) that they need to match.
In places where whitespace doesn't mattern in PHP, it has no special meaning in PPL as well.

### PHP variables

PHP variables syntax, `$<id>`

### Matcher expressions

Expressions in form of `${"<matcher>"}` or `${'<matcher>'}` are called **matcher expressions**.
The `<matcher>` determines what will be matched.

It does not matter whether you use `'` or `"`, both behave identically.

```
matcher_expr = "$" "{" quote matcher quote "}"
quote = "\"" | "'"
matcher = named_matcher | matcher_op
named_matcher = name ":" matcher_op
matcher_op = <see the table of supported ops below>
```

| Op | Description |
|---|---|
| `*` | Any node, 0-N times |
| `+` | Any node, 1-N times |
| `int` | Integer literal |
| `float` | Float literal |
| `num` | Integer or float literal |
| `str` | String literal |
| `var` | Variable |

Some examples of complete matcher expressions:
* `${'*'}` - matches any number of nodes
* `${"+"}` - matches one or more nodes
* `${'str'}` - matches any kind of string literal
* `${"x:int"}` - `x`-named matcher that matches any integer
* `$${"var"}` - matches any "variable variable", like `$$x` and `$$php`

### Filtering functions

#### `~` filter

The `~` filter matches matched node source text against regular expression.

> Note: it uses **original** source text, not the printed AST node representation.
> It means that you need to take code formatting into account.

#### `!~` filter

Opposite of `~`. Matches only when given regexp is not matched.

#### `=` filter

| Op | Effect | Example |
|---|---|---|
| `*` | Sub-pattern | `x=[$_]` |
| `+` |  Sub-pattern | `x=${"var"}` |
| `int` | Value matching | `x=1\|20\|300` |
| `float` | Value matching | `x=5.6` |
| `num` | Value matching | `x=1\|1.5` |
| `str` | Value matching | `x="foo"\|"bar"` |
| `var` | Name matching | `x=length\|len` |

Sub-pattern can include any valid PPL text.

Value and name matching accept a `|`-separated lists of permitted values.
For strings you need to use quotes, so there is no problem with having `|` inside them.

#### `!=` filter

Opposite of `=`. Matches only when `=` would not match.
