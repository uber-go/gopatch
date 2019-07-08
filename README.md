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

Note that unlike [gofmt rewrite rules], gopatch supports transformations that
span multiple lines, and affect type or function declarations.

  [gofmt rewrite rules]: https://golang.org/cmd/gofmt/

```diff
@@
@@
-client.Timeout = 5 * time.Second
+client.Timeout = 3 * time.Second
```

## Usage

    gopatch [options] [pattern...]

Where `pattern...` specifies one or more Go package patterns. The matched code
will be transformed based on the patch provided to gopatch.

Patches can be provided to gopatch on stdin or with the `-p` option. The
following two commands are equivalent.

```shell
$ gopatch -p foo.patch ./...
$ gopatch ./... < foo.patch
```

The `-p` flag is valuable when multiple patches need to be provided.

```shell
$ gopatch -p foo.patch -p bar.patch ./...
```

## Patch Files

Patch files contain one or more actual patches. Each patch starts with a
metavariables section (opened and closed with `@@`; more on this later)
followed by a unified diff specifying the transformation.

```diff
@@
@@
-foo(42)
+foo(45)

@@
@@
-foo
+bar
```

The patch file above is comprised of two patches. The first one changes all
calls of `foo` with the argument 42 to provide 45 instead. The second one
renames all instances of the identifier `foo` to `bar`. These will be run
in-order.

### Metavariables

Semantic transformations are powerful but on their own, they are limited to
matching exact identifiers and expressions. It's a common need to match any
expression or identifier. gopatch solves that with metavariables.

Metavariables are specified at the top of a patch between the `@@` symbols.

```diff
@@
...metavariables go here...
@@
-foo
+bar
```

Metavariables are declared like Go variables with `var` and can have one of the
following types.

identifier
:   Match any Go identifier.

expression
:   Match any Go expression.

For example,

```diff
@@
var x expression
@@
-foo(x)
+bar(x)
```

This replaces all calls to `foo` with calls to `bar`. The variable `x` will
match and reproduce any Go expression. This means it will work on all of the
following cases,

    foo(42)             => foo(42)
    foo(y)              => foo(y)
    foo(getValue())     => bar(getValue())
    foo(x.Value())      => bar(x.Value())

Another example is,

```diff
@@
var x expression
@@
-foo(x, x)
+v := x
+foo(v, v)
```

This will look for instances where the expression passed to `foo` is duplicated
and convert that into a variable.

    foo(getValue(), getValue())

    // becomes...

    v := getValue()
    foo(v, v)

### Patch

The patch follows the metavariables section. It is a unified diff specifying
the transformation. The diff specifies a transformation on *exactly one
instance* of any of the following,

-   function declaration

    ```diff
    @@
    @@
     func foo(
    -   uuid string,
    +   uuid UUID,
     ) {
     }
    ```

-   type declaration

    ```diff
    @@
    @@
    -type Foo struct {
    +type Bar struct {
     }
    ```

-   variable declaration

    ```diff
    @@
    @@
     var (
    -   x = 42
    +   y = 42
     )
    ```

-   constant declaration

    ```diff
    @@
    @@
     const (
    -   x = 42
    +   y = 100
     )
    ```

(If you need to transform multiple instances of any of the above, as noted
above, multiple patches may be specified in the same patch file.)

Instead of the above, the diff can also specify a transformation on one or more
statements.

```diff
# Don't log and return.
@@
var f expression
@@
 err := f()
 if err != nil {
-    log.Print(err)
     return err
 }
```

#### Package Names

If needed, package names are specified in the patch before any other Go code.

```diff
@@
@@
 package foo

-FooClient
+Client

@@
@@
-foo.FooClient
+foo.Client
```

The patch above replaces all instances of `foo.FooClient` with `foo.Client` to
reduce stuttering in its usage. The first portion of the patch only applies
inside the `foo` package.

Package names may also be changed with the diff.

```diff
@@
@@
-package foo
+package bar

-Foo
+Bar
```

The patch above renames both, the package and the type declared in that package.

## Prior Art

- [gofmt rewrite rules] support simple transformations on expressions.
- [eg] supports basic example-based refactoring.
- [Coccinelle] is a tool for C from which gopatch takes inspiration heavily.

  [eg]: https://godoc.org/golang.org/x/tools/cmd/eg
  [Coccinelle]: http://coccinelle.lip6.fr/documentation.php

## Credits

As mentioned previously, gopatch is heavily inspired by [Coccinelle]. Most
ideas for gopatch comes from [Coccinelle].
