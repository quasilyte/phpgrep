# phpgrep pattern language

This file serves as a main documentation source for the pattern language used inside phpgrep.

We'll refer to it as PPL (phpgrep pattern language) for brevity.

## Overview

The pattern language is 100% compatible with PHP syntax.

In fact, it only changes the semantics of some syntax constructions without adding any
new syntax forms to the PHP language. It means that phpgrep patterns can be parsed with
any parser that can handle PHP.

The patterns describe the program parts (syntax trees) that they need to match.
In places where whitespace doesn't mattern in PHP, it has no special meaning in PPL as well.
Unfortunately, it also means that you can't write a pattern that expects some particular
formatting (number of spaces, etc.) using PPL.

### Matcher expressions

Expression of `${"..."}` or `${'...'}` are **matcher expressions**.
The `...` determines what will be matched.
For example, `${"*"}` matches everything, 0-N times.
