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

For more on metavariables see [Metavariables](#metavariables).

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

For more on transforming statements, see [Statements](#statements).

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

# Patches in depth

This section covers the patch language in detail.

Patch files contain one or more actual patches. Each patch starts with a
metavariables section (opened and closed with `@@`) followed by a unified diff
specifying the transformation.

For example,

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

## Metavariables

Metavariables are specified at the top of a patch between the `@@` symbols.

```diff
@@
# Metavariables go here
@@
-foo
+bar
```

Metavariables are declared like Go variables with `var` and can have one of the
following types.

- [**identifier**](#identifier-metavariables): match any Go identifier
- [**expression**](#expression-metavariables): match any Go expression

> **Unclear on the difference between expressions and identifiers?**
>
> Check [Identifiers vs expressions vs statements].

  [Identifiers vs expressions vs statements]: #identifiers-vs-expressions-vs-statements

Metavariables are matched in the `-` section and if referenced in the `+`
section, the matched contents are reproduced.

### Identifier metavariables

Metavariables with the type `identifier` match and any Go identifier.
Identifiers are singular names of entities in Go. For example, in
`type Foo struct{ Bar int }`, `Foo` and `Bar` are both identifiers.

You can use identifier metavariables to capture names in your patches.

For example,

```diff
@@
var x identifier
@@
-var x = value
+x := value
```

The metavariable `x` will capture the name of variables in matching
assignments.

| Input             | `x`   | Output         |
|-------------------|-------|----------------|
| `var x = 42`      | `x`   | `x := 42`      |
| `var foo = bar()` | `foo` | `foo := bar()` |

### Expression metavariables

Metavariables with the type `expression` match any Go expression. Expressions
refer to code that has value. This includes references to variables (`foo`),
function calls (`foo()`), references to attributes of variables (`foo.Bar`),
and more.

You can use expression metavariables to capture arbitrary Go expressions.

For example,

```diff
@@
var x expression
@@
-foo(x)
+bar(x)
```

The metavariable `x` will capture any argument to a `foo` function call, no
matter how complex.

| Input             | `x`          | Output            |
|-------------------|--------------|-------------------|
| `foo(42)`         | `42`         | `bar(42)`         |
| `foo(y)`          | `y`          | `bar(y)`          |
| `foo(getValue())` | `getValue()` | `bar(getValue())` |
| `foo(x.Value())`  | `x.Value()`  | `bar(x.Value())`  |

### Metavariable repetition

If the same metavariable appears multiple times in the `-` section of the
patch, occurrences after the first are expected to match the previously
recorded values.

For example,

```diff
@@
var x expression
@@
-foo(x, x)
+v := x
+foo(v, v)
```

The above will only match cases where both arguments to `foo` are *exactly*
the same.

| Input                         | Match |
|-------------------------------|-------|
| `foo(a, a)`                   | Yes   |
| `foo(x, y)`                   | No    |
| `foo(getValue(), getValue())` | Yes   |

## Diff

In a patch, the diff section follows the metavariables. This section is where
you specify the code transformation.

The diff section **optionally** begins with the following:

- [Package name](#package-names)
- [Imports](#imports)

Note that these are optional. If you don't wish to match on or modify the
package name or imports, you can omit them.

Following the package name and imports, if any, the diff specifies a
transformation on **exactly one** of the following:

- [Expressions](#expressions)
- [Statements](#statements)
- [Function declarations](#function-declarations)
- [Type declarations](#type-declarations)

> Support for multiple transformations in the same diff will be added in a
> future version of gopatch. Meanwhile, you may specify multiple patches in
> the same file. See also [#4]

  [#4]: https://github.com/uber-go/gopatch/issues/4

### Package Names

gopatch supports matching on, and manipulating package names.

- [Matching package names](#matching-package-names)
- [Renaming packages](#renaming-packages)

#### Matching package names

Package names may be specified at the top of the diff similar to Go code.

```diff
@@
@@
 package foo

# Rest of the diff
```

If specified, the diff will apply only to files with that package name.

For example, following patch renames `foo.FooClient` to `foo.Client` to reduce
stuttering in its usage. (See [Avoid stutter] for the motivation for this
change.)

<!-- This is used in the imports example below to reference this position.-->
<a name="avoid-stutter-patch"></a>

  [Avoid stutter]: https://blog.golang.org/package-names#TOC_3.

```diff
@@
@@
 package foo

-FooClient
+Client
```

Note that this patch does not yet update consumers of
`foo.FooClient`. Check the [Imports](#imports) section for how to do that.

#### Renaming packages

Package clauses can also be prefixed with `-` or `+` to rename packages as
part of the patch.

```diff
@@
@@
-package foo
+package bar

# rest of the diff
```

For example, the following patch renames the package and an object defined in
it.

```diff
@@
@@
-package foo
+package bar

-Foo
+Bar
```

Again, as with the previous patch, this does not rename consumers.

### Imports

gopatch allows matching on, and manipulating imports in a file.

- [Matching imports](#matching-imports)
- [Matching any import](#matching-any-import)
- [Changing imports](#changing-imports)
- [Changing any import](#changing-any-import)
- [Best practices for imports](#best-practices-for-imports)

#### Matching imports

Imports appear at the top of the diff, after the package clause (if any).

```diff
@@
@@
 import "example.com/bar"

 # rest of the patch

@@
@@
-package foo
+package bar

 import "example.com/bar"

# rest of the diff
```

Imports may be unnamed, like the patch above, or they may be named like the
following.

```diff
@@
@@
 import mybar "example.com/bar"

# rest of the diff
```

These imports are matched exactly as-is. That is, the unnamed import will only
match files which import the package unnamed, and the named import will only
match files that import the package with that exact name.

<table>
<thead><tr><th>Patch</th><th>Input</th><th>Matches</th></tr></thead>
<tbody>
<tr><td>

```diff
@@
@@
 import "example.com/bar"

# ...
```

</td><td>

```go
package foo

import "example.com/bar"
```

</td><td>

Yes

</td></tr>
<tr><td>

```diff
@@
@@
 import mybar "example.com/bar"

# ...
```

</td><td>

```go
package foo

import mybar "example.com/bar"
```

</td><td>

Yes

</td></tr>
<tr><td>

```diff
@@
@@
 import "example.com/bar"

# ...
```

</td><td>

```go
package foo

import mybar "example.com/bar"
```

</td><td>

No

</td></tr>
<tr><td>

```diff
@@
@@
 import mybar "example.com/bar"

# ...
```

</td><td>

```go
package foo

import notmybar "example.com/bar"
```

</td><td>

No

</td></tr>
</tbody></table>

#### Matching any import

gopatch supports matching all imports of a specific import path, named or
unnamed. To do this, declare an `identifier` [metavariable](#metavariables)
and use that as the named import in the diff.

```diff
@@
var bar identifier
@@
 import bar "example.com/bar"

# rest of the patch
```

As a complete example, building upon the [patch above to avoid stuttering], we
can now update consumers of `foo.FooClient`.

  [patch above to avoid stuttering]: #avoid-stutter-patch

```diff
@@
@@
 import "example.com/foo"

-foo.FooClient
+foo.Client

@@
var foo identifier
@@
 import foo "example.com/foo"

-foo.FooClient
+foo.Client
```

The first diff in this patch affects files that use unnamed imports, and the
second affects those that use named imports---regardless of name.

> *Note*: In a future version of gopatch, we'll need only the second patch to
> make this transformation. See also, [#2].

  [#2]: https://github.com/uber-go/gopatch/issues/2

#### Changing imports

In addition to matching on imports, you can also change imports with gopatch.
For example,

```diff
@@
@@

-import "example.com/foo"
+import "example.com/bar"

-foo.Client
+bar.Client
```

> *Note*: It's a known limitation in gopatch right now that there must be
> something after the `import`. You cannot currently write patches that only
> match and change imports. See [#5] for more information.
>
> Meanwhile, you can work around this by writing a patch which matches but
> does not change an arbitrary identifier in the imported package. For
> example,
>
> ```diff
> @@
> var x identifier
> @@
> -import "example.com/foo"
> +import "example.com/internal/foo"
>
> foo.x
> ```
>
> This will match files that import `example.com/foo` and have at least one
> reference to *anything* in that package.

  [#5]: https://github.com/uber-go/gopatch/issues/5

You can match on and manipulate, both, named and unnamed imports.

For example, the following patch will search for an unnamed imports of a
specific package and turn those into named imports.

```diff
@@
var x identifier
@@
-import "example.com/foo-go.git"
+import foo "example.com/foo-go.git"

 foo.x
```

(It's good practice in Go to use a named import when the last component of the
import path, `foo-go.git` in this example, does not match the package name,
`foo`.)

#### Changing any import

As with [matching any import](#matching-any-import), you can declare an
identifier metavariable to match and manipulate both, named and unnamed
imports.

```diff
@@
var foo, x identifier
@@
-import foo "example.com/foo-go.git"
+import foo "example.com/foo.git"

 foo.x
```

The above will match, both, named and unnamed imports of
`example.com/foo-go.git` and change them to imports of `example.com/foo.git`,
*preserving the name of a matched import*.

| Input                                 | Output                              |
|---------------------------------------|-------------------------------------|
| `import foo "example.com/foo-go.git"` | `import foo "example.com/foo.git"`  |
| `import bar "example.com/foo-go.git"` | `import bar "example.com/foo.git"`  |
| `import "example.com/foo-go.git"`     | `import foo "example.com/foo.git"`* |

> *This case is a known bug. See [#2] for more information.
>
> You can work around this by first explicitly matching and replacing the
> cases with unnamed imports first. For example, turn the patch above into two
> diffs, one addressing the unnamed imports, and one addressing the named.
>
> ```diff
> @@
> var x identifier
> @@
> -import "example.com/foo-go.git"
> +import "example.com/foo.git"
>
>  foo.x
>
> @@
> var foo, x identifier
> @@
> -import foo "example.com/foo-go.git"
> +import foo "example.com/foo.git"
>
>  foo.x
> ```

#### Best practices for imports

Given the known limitations and issues with imports highlighted above, the
best practices for matching and manipulating imports are:

- Handle unnamed imports first. This will make sure that previously named
  imports do not unintentionally become named.
- When matching any import, use a metavariable name that matches the name of
  the imported package **exactly**. This name is used by gopatch to guess the
  name of the package.

    ```
    # BAD                           | # GOOD
    @@                              | @@
    var x identifier                | var foo identifier
    @@                              | @@
     import x "example.com/foo.git" |  import foo "example.com/foo.git"
    ```

### Expressions

gopatch can match and transform expressions. This is the most basic type of
transformation. These appear after the [package name](#package-names) and
[imports](#imports) (if any).

> **Unclear on the difference between expressions and statements?**
>
> Check [Identifiers vs expressions vs statements].

```diff
@@
@@
-user.GetName()
+user.Name
```

Expression transformations can use [metavariables](#metavariables) to specify
which parts of them should be generic.

```diff
@@
var x identifier
@@
-fmt.Sprintf("%v", x)
+fmt.Sprint(x)
```

### Statements

gopatch can match and transform statements.
These appear after the [package name](#package-names) and [imports](#imports) (if any).

> **Unclear on the difference between expressions and statements?**
>
> Check [Identifiers vs expressions vs statements].

```diff
@@
@@
-var x string = y
+x := y
```

Statement transformations may use [metavariables](#metavariables).

```diff
@@
var err identifier
var log expression
@@
 if err != nil {
-   log.Error(err)
    return err
 }
```

### Function declarations

gopatch can match and modify function declarations. These appear after the
[package name](#package-names) and [imports](#imports) (if any).


```diff
@@
@@
 func foo(
-   uuid string,
+   uuid UUID,
 ) {
  ...
 }
```

This works for functions with receivers too.

```diff
@@
var t identifier
var T expression
@@
 func (t *T) String() string {
+  if t == nil {
+    return "<nil>"
+  }
   ...
 }
```

### Type declarations

gopatch can match and modify type declarations. These appear after the
[package name](#package-names) and [imports](#imports) (if any).

```diff
@@
@@
 type User struct {
-  UserName string
+  Name string
 }
```

Transformations on type declarations can use `identifier` metavaribles to
capture names of types and fields, and `expression` metavariables to capture
field types.

```diff
@@
var A, B identifier
var Type expression
@@
 type Config struct {
-   A Type
-   B Type
+   A, B Type
 }
```

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
