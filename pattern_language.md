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

### Matcher expressions

Expression of `${"..."}` or `${'...'}` are **matcher expressions**.
The `...` determines what will be matched.
For example, `${"*"}` matches everything, 0-N times.

It does not matter whether you use `'` or `"`, both behave identically.

| Syntax | Examples | Description |
|---|---|---|
| `\*` | `${'*'}` | Matches any nodes, 0-N times. |
| `\+` | `${'+'}` | Matches any nodes, 1-N times. |
| `\$.*` | `${'$.*'}` `${'$.*_id'}` | Matches variable that matches the associated regexp. |
