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

Unlike [gofmt rewrite rules], gopatch supports transformations that span
multiple lines, and affect type or function declarations.

  [gofmt rewrite rules]: https://golang.org/cmd/gofmt/

```diff
@@
@@
-client.Timeout = 5 * time.Second
+client.Timeout = 3 * time.Second
```

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
