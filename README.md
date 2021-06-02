# gopatch

gopatch is a tool to match and transform Go code. It is meant to aid in
refactoring and restyling.

# Table of contents

- [Introduction](#introduction)
- [Getting started](#getting-started)
  - [Installation](#installation)
  - [Your first patch](#your-first-patch)
  - [Apply the patch](#apply-the-patch)
  - [Next steps](#next-steps)
- [Usage](#usage)
  - [Options](#options)
- [Patches](#patches)
  - [Metavariables](#metavariables)
  - [Statements](#statements)
  - [Elision](#elision)
- [Examples](#examples)
- [Project status](#project-status)
  - [Goals](#goals)
  - [Known issues](#known-issues)
  - [Upcoming](#upcoming)
- [Similar Projects](#similar-projects)
- [Credits](#credits)

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

To learn how to write your own patches, move on to the [Patches] section. To
dive deeper into patches, check out [Patches in depth].

  [Patches in depth]: docs/PatchesInDepth.md

To experiment with other sample patches, check out the [Examples] section.

  [Patches]: #patches
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
    patch file format in [Patches].

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

# Patches

Patch files are the input to gopatch that specify how to transform code. Each
patch file contains one or more patches. This section provides an introduction
to writing patches; look at [Patches in depth] for a more detailed
explanation.

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

## Metavariables

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
> Expressions usually refer to code that has value. You can pass these as
> arguments to functions. These include `x`, `foo()`, `user.Name`, etc.
>
> Check the [Identifiers vs expressions vs statements] section of the appendix
> for more.

  [Identifiers vs expressions vs statements]: docs/Appendix.md#identifiers-vs-expressions-vs-statements

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

For more on metavariables see [Patches in depth/Metavariables].

  [Patches in depth/Metavariables]: docs/PatchesInDepth.md#metavariables

## Statements

gopatch patches are not limited to transforming basic expressions. You can
also transform statements.

> **What is a Go statements?**
>
> Statements are instructions to do things, and do not have value. They cannot
> be passed as parameters to other functions. These include assignments
> (`foo := bar()`), if statements (`if foo { bar() }`), variable declarations
> (`var foo Bar`), and so on.
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

For more on transforming statements, see [Patches In Depth/Statements].

  [Patches In Depth/Statements]: docs/PatchesInDepth.md#statements

## Elision

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

Going back to the patch from [Statements], we can instead write the following
patch.

  [Statements]: #statements

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

For more on elision, see [Patches in depth/Elision].

  [Patches in depth/Elision]: docs/PatchesInDepth.md#elision

# Examples

This section lists various example patches you can try in your code.
Note that some of these patches are not perfect and may have false positives.

- [s1012.patch](examples/s1012.patch): Fix for staticcheck [S1012](https://staticcheck.io/docs/checks#S1012).
- [s1028.patch](examples/s1028.patch): Fix for staticcheck [S1028](https://staticcheck.io/docs/checks#S1028).
- [s1038.patch](examples/s1038.patch): Fix for staticcheck [S1038](https://staticcheck.io/docs/checks#S1038).
- [gomock-v1.5.0.patch](examples/gomock-v1.5.0.patch): Drops unnecessary call to `Finish` method for users of gomock.
- [destutter.patch](examples/destutter.patch): Demonstrates renaming a type and updating its consumers.

# Project status

The project is currently is in a beta state. It works but significant features
are planned that may result in breaking changes to the patch format.

## Goals

gopatch aims to be a generic power tool that you can use in lieu of simple
search-and-replace.

gopatch will attempt to do 80% of the work for you in a transformation, but it
cannot guarantee 100% correctness or completeness. Part of this is owing to
the decision that gopatch must be able to operate on code that doesn't yet
compile, which can often be the case in the middle of a refactor. We may add
features in the future that require compilable code, but we plan to always
support transformation of partially-valid Go code.

## Known issues

Beyond the known issues highlighted above, there are a handful of other issues
with using gopatch today.

- It's very quiet, so there's no indication of progress. [#7]
- Error messages for invalid patch files are hard to decipher. [#8]
- Matching elisions between the `-` and `+` sections does not always work in a
  desirable way. We may consider replacing anonymous `...` elision with a
  different named elision syntax to address this issue. [#9]
- When elision is used, gopatch stops replacing after the first instance in
  the given scope which is often not what you want. [#10]
- Formatting of output generated by gopatch isn't always perfect.

  [#7]: https://github.com/uber-go/gopatch/issues/7
  [#8]: https://github.com/uber-go/gopatch/issues/8
  [#9]: https://github.com/uber-go/gopatch/issues/9
  [#10]: https://github.com/uber-go/gopatch/issues/10

## Upcoming

Besides addressing the various limitations and issues we've already mentioned,
we have a number of features planned for gopatch.

- Contextual matching: match context (like a function declaration), and then
  run a transformation inside the function body repeatedly, at any depth. [#11]
- Collateral changes: Match and capture values in one patch, and use those in
  a following patch in the same file.
- Metavariable constraints: Specify constraints on metavariables, e.g.
  matching a string, or part of another metavariable.
- Condition elision: An elision should match only if a specified condition is
  also true.

  [#11]: https://github.com/uber-go/gopatch/issues/11

# Similar Projects

- [rf] is a refactoring tool with a custom DSL
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
  [rf]: https://github.com/rsc/rf

# Credits

gopatch is heavily inspired by [Coccinelle].
