# gopatch

gopatch is a tool to match and transform Go code. It is meant to aid in
refactoring and restyling.

# Introduction

gopatch operates like the Unix `patch` tool: given a patch file and another
file as input, it applies the changes specified in the patch to the provided
file.

```
 .-------.                      .-------.
/_|      |.                    /_|      |.
|        ||.    +---------+    |        ||.
|   .go  |||>-->| gopatch |>-->|   .go  |||
|        |||    +---------+    |        |||
'--------'||      ^            '--------'||
 '--------'|      |             '--------'|
  '--------'      |              '--------'
     .-------.    |
    /_|      |    |
    |        +----'
    | .patch |
    |        |
    '--------'
```

What specifically differentiates it from `patch` is that unlike plain text
transformations, it can be smarter because it understands Go syntax.

# Getting started

## Installation

Install gopatch with the following command.

```bash
go install github.com/uber-go/gopatch@latest
```

## Your first patch

Write your first patch.

```shell
$ cat > ~/s1028.patch
@@
@@
-import "errors"

-errors.New(fmt.Sprintf(...))
+fmt.Errorf(...)
```

This patch is a fix for staticcheck [S1028]. It searches for uses of
[`fmt.Sprintf`] with [`errors.New`], and simplifies them by replacing them
with [`fmt.Errorf`].

  [S1028]: https://staticcheck.io/docs/checks#S1028
  [`fmt.Sprintf`]: https://golang.org/pkg/fmt/#Sprintf
  [`errors.New`]: https://golang.org/pkg/errors/#New
  [`fmt.Errorf`]: https://golang.org/pkg/fmt/#Errorf

For example,

```go
return errors.New(fmt.Sprintf("invalid port: %v", err))
// becomes
return fmt.Errorf("invalid port: %v", err)
```

## Apply the patch

To apply the patch, `cd` to your Go project's directory.

```shell
$ cd ~/go/src/example.com/myproject
```

Run `gopatch` on the project, supplying the previously written patch with the
`-p` flag.

```shell
$ gopatch -p ~/s1028.patch ./...
```

This will apply the patch on all Go code in your project.

Check if there were any instances of this issue in your code by running
`git diff`.

## Next steps

To learn how to write your own patches, move on to the [Introduction to
patches] section.

To experiment with other sample patches, check out the [Examples] section.

  [Introduction to patches]: #introduction-to-patches
  [Examples]: #examples

# Usage

To use the gopatch command line tool, provide the following arguments.

```
gopatch [options] pattern ...
```

Where pattern specifies one or more Go files, or directories containing Go
files. For directories, all Go code inside them and their descendants will be
considered by gopatch.

## Options

gopatch supports the following command line options.

- `-p file`, `--patch=file`

    Path to a patch file specifying a transformation. Read more about the
    patch file format in [Introduction to patches].

    Provide this flag multiple times to apply multiple patches in-order.

    ```shell
    $ gopatch -p foo.patch -p bar.patch path/to/my/project
    ```

    If this flag is omitted, a patch is expected on stdin.

    ```shell
    $ gopatch path/to/my/project << EOF
    @@
    @@
    -foo
    +bar
    EOF
    ```

# Introduction to patches

Patch files are the input to gopatch that specify how to transform code. Each
patch file contains one or more patches.

Each patch specifies a code transformation. These are formatted like unified
diffs: lines prefixed with `-` specify matching code should be deleted, and
lines prefixed with `+` specify that new code should be added.

Consider the following patch.

```diff
@@
@@
-foo
+bar
```

It specifies that we want to search for references to the identifier `foo` and
replace them with references to `bar`. (Ignore the lines with `@@` for now.
We will cover those below.)

A more selective version of this patch will search for uses of `foo` where it
is called as a function with specific arguments.

```diff
@@
@@
-foo(42)
+bar(42)
```

This will search for invocations of `foo` as a function with the specified
argument, and replace only those with `bar`.

gopatch understands Go syntax, so the above is equivalent to the following.

```diff
@@
@@
-foo(
+bar(
  42,
 )
```

## Introduction to metavariables

Searching for hard-coded exact parameters is limited. We should be able to
generalize our patches.

The previously ignored `@@` section of patches is referred to as the
**metavariable section**. That is where we specify **metavariables** for the
patch.

Metavariables will match any code, to be reproduced later. Think of them like
holes to be filled by the code we match. For example,

```diff
@@
var x expression
@@
# rest of the patch
```

This specifies that `x` should match any Go expression and record its match
for later reuse.

> **What is a Go expression?**
>
> Expressions refer to code that has value. You can pass theses as arguments
> functions. These include `x`, `foo()`, `user.Name`, etc.
>
> Check the [Identifiers vs expressions vs statements] section of the appendix
> for more.

So the following patch will search for invocations of `foo` with a single
argument---any argument---and replace them with invocations of `bar` with the
same argument.

```diff
@@
var x expression
@@
-foo(x)
+bar(x)
```

| Input              | Output             |
|--------------------|--------------------|
| `foo(42)`          | `bar(42)`          |
| `foo(answer)`      | `bar(answer)`      |
| `foo(getAnswer())` | `bar(getAnswer())` |


Metavariables hold the entire matched value, so we can add code around them
without risk of breaking anything.

```diff
@@
var x expression
@@
-foo(x)
+bar(x + 3, true)
```

| Input              | Output                       |
|--------------------|------------------------------|
| `foo(42)`          | `bar(42 + 3, true)`          |
| `foo(answer)`      | `bar(answer + 3, true)`      |
| `foo(getAnswer())` | `bar(getAnswer() + 3, true)` |

## Introduction to statement transformations

gopatch patches are not limited to transforming basic expressions. You can
also transform statements.

> **What is a Go expression?**
>
> Statements are instructions that do not have value. They cannot be passed as
> parameters to other functions. These include assignments (`foo := bar()`),
> if statements (`if foo { bar() }`), variable declarations (`var foo Bar`),
> and so on.
>
> Check the [Identifiers vs expressions vs statements] section of the appendix
> for more.

For example, consider the following patch.

```diff
@@
var f expression
var err identifier
@@
-err = f
-if err != nil {
+if err := f; err != nil {
   return err
 }
```

The patch declares two metavariables:

- `f`: This represents an operation that possibly returns an `error`
- `err`: This represents the name of the `error` variable

The patch will search for code that assigns to an error variable immediately
before returning it, and inlines the assignment into the `if` statement. This
effectively [reduces the scope of the variable] to just the `if` statement.

  [reduces the scope of the variable]: https://github.com/uber-go/guide/blob/master/style.md#reduce-scope-of-variables

<table>
<thead><tr><th>Input</th><th>Output</th></tr></thead>
<tbody>
<tr><td>

```go
err = foo(bar, baz)
if err != nil {
   return err
}
```

</td><td>

```go
if err := foo(bar, baz); err != nil {
   return err
}
```

</td></tr>
<tr><td>

```go
err = comment.Submit(ctx)
if err != nil {
  return err
}
```

</td><td>

```go
if err := comment.Submit(ctx); err != nil {
  return err
}
```

</td></tr>
</tbody></table>

## Introduction to elision

Matching a single argument is still too selective and we may want to match a
wider criteria.

For this, gopatch supports **elision** of code by adding `...` in many places.
For example,

```diff
@@
@@
-foo(...)
+bar(...)
```

The patch above looks for all calls to the function `foo` and replaces them
with calls to the function `bar`, regardless of the number of arguments they
have.

| Input                      | Output                     |
|----------------------------|----------------------------|
| `foo(42)`                  | `bar(42)`                  |
| `foo(42, true, 1)`         | `bar(42, true, 1)`         |
| `foo(getAnswer(), x(y()))` | `bar(getAnswer(), x(y()))` |

Going back to the patch from [statement transformations], we can instead write
the following patch.

  [statement transformations]: #introduction-to-statement-transformations

```diff
@@
var f expression
var err identifier
@@
-err = f
-if err != nil {
+if err := f; err != nil {
   return ..., err
 }
```

This patch is almost exactly the same as before except the `return` statement
was changed to `return ..., err`. This will allow the patch to operate even on
functions that return multiple values.

<table>
<thead><tr><th>Input</th><th>Output</th></tr></thead>
<tbody>
<tr><td>

```go
err = foo()
if err != nil {
   return false, err
}
```

</td><td>

```go
if err := foo(); err != nil {
   return false, err
}
```

</td></tr>
</tbody></table>

# Similar Projects

- [gofmt rewrite rules] support simple transformations on expressions
- [eg] supports basic example-based refactoring
- [Coccinelle] is a tool for C from which gopatch takes inspiration heavily
- [Semgrep] is a cross-language semantic search tool
- [Comby] is a language-agnostic search and transformation tool

  [gofmt rewrite rules]: https://golang.org/cmd/gofmt/
  [eg]: https://godoc.org/golang.org/x/tools/cmd/eg
  [Coccinelle]: https://coccinelle.gitlabpages.inria.fr/website/
  [Semgrep]: https://semgrep.dev/
  [Comby]: https://comby.dev/

# Credits

As mentioned previously, gopatch is heavily inspired by [Coccinelle]. Most
ideas for gopatch comes from [Coccinelle].

# Appendix

## Identifiers vs expressions vs statements

A simplified explanation of the difference between identifiers, expressions
and statements is,

- **identifiers** are names of things
- **expressions** are things that have values (you can pass these into functions)
- **statements** are instructions to do things

Consider the following snippet.

```go
if err := foo(bar.Baz()); err != nil {
  return err
}
```

It contains,

- identifiers: `err`, `foo`, `bar`, `Baz`

    ```
    if err := foo(bar.Baz()); err != nil {
       '-'    '-' '-' '-'     '-'
      return err
             '-'
    }
    ```

- expressions: `bar`, `bar.Baz`, `bar.Baz()`, `foo(bar.Baz())`, `err`, `nil`,
  and `err != nil`

    ```
                  .-------.
              .---|-------|.  .---------.
    if err := foo(bar.Baz()); err != nil {
                  '-'   |     '-'    '-'
                  '-----'
      return err
             '-'
    }
    ```

- statements: `err := ...`, `if ...`, and `return ...`

    ```
    if err := foo(bar.Baz()); err != nil {  -.
       '-------------------'                 |
      return err                             |
      '--------'                             |
    }                                       -'
    ```
