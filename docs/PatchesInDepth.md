# Table of contents

- [Metavariables](#metavariables)
  - [Identifier metavariables](#identifier-metavariables)
  - [Expression metavariables](#expression-metavariables)
  - [Metavariable repetition](#metavariable-repetition)
- [Diff](#diff)
  - [Package Names](#package-names)
  - [Imports](#imports)
  - [Expressions](#expressions)
  - [Statements](#statements)
  - [Function declarations](#function-declarations)
  - [Type declarations](#type-declarations)
  - [Value declarations](#value-declarations)
- [Elision](#elision)
- [Grammar](#grammar)

# Patches in depth

This document covers the patch language in detail.

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

  [Identifiers vs expressions vs statements]: Appendix.md#identifiers-vs-expressions-vs-statements

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
refer to code that has value or refers to a type. This includes references to
variables (`foo`), function calls (`foo()`), references to attributes of
variables (`foo.Bar`), type references (`[]foo`), and more.

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
- [Value declarations](#value-declarations)

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

#### Elision in expressions

Expressions support [elision](#elision) on various components.

- [Function call parameters](#function-call-parameters)
- [Struct fields](#struct-fields)
- [Slice elements](#slice-elements)
- [Map items](#map-items)
- [Anonymous functions](#anonymous-functions)

##### Function call parameters

```diff
@@
@@
-f(...)
+g(...)

@@
@@
-f(ctx, ...)
+g(ctx, ...)

@@
@@
 f(
  ...,
- user,
+ user.Name,
  ...,
)
```

##### Struct fields

```diff
@@
@@
-Foo{...}
+Bar{...}

@@
@@
 User{
    ...,
-   UserName: value,
+   Name: value,
    ...,
  }
```

##### Slice elements

```diff
@@
@@
-[]string{...}
+[]Email{...}

@@
@@
 []string{
   ...,
-  "foo",
+  _foo,
   ...,
 }
```

##### Map items

```diff
@@
@@
-map[string][]string{...}
+http.Header{...}

@@
var v expression
@@
 map[string]string{
   ...,
-  "foo": "bar",
   ...,
 }
```

##### Anonymous functions

```diff
@@
-func() {
+func(context.Context) {
    ...
 }
```

Anonymous functions are a special case of elision in
[function declarations](#elision-in-function-declarations).

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

#### Elision in statements

A few different kinds of statements support [elision](#elision) with `...`.

- [Statement blocks](#statement-blocks)
- [For and range statements](#for-and-range-statements)
- [Return statements](#return-statements)

##### Statement blocks

<a name="elision-statement-blocks"></a>

```diff
@@
var t expression
var ctrl identifier
@@
 ctrl := gomock.NewController(t)
 ...
-defer ctrl.Finish()
```

These may be inside other statements.

```diff
@@
var err identifier
var log expression
@@
  if err != nil {
    ...
-   log.Error(err)
    return err
  }
```

##### For and range statements

```diff
@@
var s identifier
var x expression
@@
-var s string
+var sb strings.Builder
 for ... {
-   s += x
+   sb.WriteString(x)
 }
+s := sb.String()
```

This will match all of the following forms of `for` statements.

```
for cond { ... }
for i := 0; i < N; i++ { ... }
for x := range items { ... }
for i, x := range items { ... }
```

##### Return statements

```diff
@@
@@
 if err != nil {
-   return ..., nil
+   return ..., err
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

#### Elision in function declarations

gopatch supports [elision](#elision) in function declarations with `...`.

- [Function bodies](#function-bodies)
- [Parameters](#parameters)
- [Receivers](#receivers)
- [Return values](#return-values)

##### Function bodies

This is the same as [elision of statement blocks](#elision-statement-blocks).

```diff
@@
@@
-func foo() {
+func foo(context.Context) {
    ...
 }
```

##### Parameters

###### Anonymous parameters

```diff
@@
var f identifier
@@
-func f(...) error {
+func f(context.Context, ...) error {
   ...
 }
```

###### Named parameters

```diff
@@
var req, send identifier
@@
 func send(
+   ctx context.Context,
    ...,
    req *http.Request,
    ...,
) error {
+   req = req.WithContext(ctx)
   ...
 }
```

##### Receivers

```diff
@@
var req identifier
@@
-func (...) Send(req *Request) error }
+func (...) SendRequest(req *Request) error {
   ...
 }
```

##### Return values

###### Anonymous return values

```diff
@@
var f identifier
@@
-func f() (error, ...) {
+func f() (..., error) {
  ...
 }
```

###### Named return values

```diff
@@
var f identifier
@@
-func f() (err error, ...) {
+func f() (..., err error) {
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

#### Elision in type declarations

gopatch supports [elision](#elision) in typedeclarations with `...`.

- [Struct fields](#struct-fields)
- [Interface methods](#interface-methods)

##### Struct fields

```diff
@@
var Ctx identifier
@@
 type Request struct {
    ...
-   Ctx context.Context
    ...
 }
```

##### Interface methods

```diff
@@
@@
 type Doer interface {
   ...
-  Do()
+  Do() error
   ...
 }
```

### Value declarations

gopatch can match and modify value declarations: both, `var` and `const`
declarations. These appear after the [package name](#package-names) and
[imports](#imports) (if any).

```diff
@@
@@
-var foo = v
+var bar = v

@@
@@
-const foo = v
+const bar = v
```

These declarations can change the kind of declaration from `var` to `const` or
vice-versa.

```diff
@@
@@
-var foo = 42
+const foo = 42
```

Transformations can operate on values inside groups as well.

```diff
@@
@@
 var (
-  foo = 43
   bar = 42
+  foo = bar + 1
 )
```

> *Note*: gopatch is currently limited to operating on the format specified in
> the patch only. That is, if the patch used `var name = value`, gopatch will
> not currently operate on `var (name = value)`. We plan to fix this in a
> future version. See [#3] for more information.

  [#3]: https://github.com/uber-go/gopatch/issues/3

Value declarations use [`identifier` metavariables] to capture names of
values, and [`expression` metavariables] to capture the values associated with
those declarations.

  [`identifier` metavariables]: #identifier-metavariables
  [`expression` metavariables]: #expression-metavariables


```diff
@@
var name identifier
var value expression
@@
-const name = value
+var name = value
```

> Note that gopatch does not yet support elision in value declarations. See
> [#6] for more information.

  [#6]: https://github.com/uber-go/gopatch/issues/6

## Elision

gopatch supports elision by adding `...` in several places to support omitting
unimportant portions of a patch.

Elisions may be added in the following places:

- [Expressions](#elision-in-expressions)
- [Statements](#elision-in-statements)
- [Function declarations](#elision-in-function-declarations)
- [Type declarations](#elision-in-type-declarations)

Elisions in the `-` and `+` sections are matched with each other based on
their positions. This doesn't always work as expected. While we plan to
address this in a future version, meanwhile you can work around this by
restructuring your patch so that elisions are on their own lines with a ` `
prefix.

For example,

<table>
<thead><tr><th>Before</th><th>After</th></tr></thead>
<tbody>
<tr><td>

```diff
@@
@@
-foo(...)
+bar(...)
```

</td><td>

```diff
@@
@@
-foo(
+bar(
   ...,
 )
```

</td></tr>
</tbody></table>

## Grammar


A file consists of one or more patches.

```
file = patch+
```

A patch consists of a metavariables section and a diff.

```
patch = metavariables diff
```

The metavariables section opens and closes with @@. It specifies zero or more
metavariables.

```
metavariables =
    '@@'
    metavariable*
    '@@'
```

Metavariables are declared in Go's 'var' declaration form.


```
metavariable =
    'var' identi metavariable_type
```

Their names must be [valid Go identifiers], and their types must be one of
`expression` and `identifier`.

  [valid Go identifiers]: https://golang.org/ref/spec#Identifiers

```
metavariable_name = identifier
metavariable_type = 'expression' | 'identifier'
```

Diffs contains lines prefixed with '-' or '+' to indicate that they represent
code that should be deleted or added, or lines prefixed with ' ' to indicate
that code they match should be left unchanged.

```
diff
    = '-' line
    | '+' line
    | ' ' line
```

The minus and plus sections of a diff form two files. For example, the
following diff,

```diff
-x, err := foo(...)
+x, err := bar(...)
 if err != nil {
   ...
-  return err
+  return nil, err
 }
```

Becomes two files:

<table>
<thead><tr><th>Find</th><th>Replace</th></tr></thead>
<tbody>
<tr><td>

```go
x, err := foo(...)
if err != nil {
  ...
  return err
}
```

</td><td>

```go
x, err := bar(...)
if err != nil {
  ...
  return nil, err
}
```

</td></tr>
</tbody></table>

Both these versions should be *almost* valid Go code with the following
exceptions:

- package clause may be omitted
- imports may be omitted
- function declarations may be omitted
- [elisions](#elision) may appear in several places
