# gopatch

gopatch is a tool that matches and manipulates Go code semantically rather than
simple string matching.

## Introduction

Transformations to Go code are specified as unified diffs.

```diff
@@
@@
-foo(1, 2, 3)
+bar(1, 2, 3)
```

The patch above will search for all invocations of the function `foo` with the
specified arguments and replace them with invocations of `bar`.

gopatch understands Go syntax so the patch above can also be expressed as
follows.

```diff
@@
@@
-foo(
+bar(
   1, 2, 3,
 )
```

Note that as per the Unified Diff format, lines must begin with `-` or `+` to
specify that the matching code must be deleted or added.

Unlike [gofmt rewrite rules], gopatch supports transformations that span
multiple lines, and affect type or function declarations.

  [gofmt rewrite rules]: https://golang.org/cmd/gofmt/

```diff
@@
@@
-client.Timeout = 5 * time.Second
+client.Timeout = 3 * time.Second
```

## Metavariables

The functionality demonstrated so far is powerful but it is limited by matching
expressions and names verbatim. Metavariables solve that.

Metavariables allow you to declare zero or more identifiers in the patch that
should be treated more generically when being matched. Metavariables are
declared in the `@@` section at the start of each change.

Consider the previous example of converting calls of `foo` to `bar`. In the
example, we were hard-coding the arguments like so, `foo(1, 2, 3)`. This won't
match all calls to `foo`. To match all calls, we declare three metavariables
for the three arguments and specify a transformation based on that.

```diff
@@
var x, y, z expression
@@
-foo(x, y, z)
+bar(x, y, z)
```

The above will transform all calls of `foo` to `bar` for *any* expressions `x`,
`y`, and `z`. This includes the following:

```
foo(1, 2, 3)  // => bar(1, 2, 3)
foo(4, 5, 6)  // => bar(4, 5, 6)
foo(a+1, rand.Int()+42, c*2)
              // => bar(a+1, rand.Int()+42, c*2)
```

Similarly, the prior example of changing the client timeout can be altered to
work for any variable instead of just `client`.

```diff
@@
var client identifier
@@
-client.Timeout = 5 * time.Second
+client.Timeout = 3 * time.Second
```

The above will look for any statement where we're setting a `Timeout` field on
an object to 5 seconds and change it to 3 seconds.

The following types of metavariables are supported:

- **expression** matches any Go expression. This includes variables, function
  calls, arithmetic expressions, etc. If you can assign it to a variable,
  expression can match it.
- **identifier** matches Go identifiers. This is useful when the name of a
  variable, type, field, or function is dynamic.

## Limitations

- Changes affecting type or function declarations can match against single
  declarations only. To match against multiple types or functions, specify
  multiple changes in the same patch.

## Prior Art

- [gofmt rewrite rules] support simple transformations on expressions.
- [eg] supports basic example-based refactoring.
- [Coccinelle] is a tool for C from which gopatch takes inspiration heavily.

  [eg]: https://godoc.org/golang.org/x/tools/cmd/eg
  [Coccinelle]: http://coccinelle.lip6.fr/documentation.php

## Credits

As mentioned previously, gopatch is heavily inspired by [Coccinelle]. Most
ideas for gopatch comes from [Coccinelle].
