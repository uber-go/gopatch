This attempts to document information necessary to hack on gopatch.

# Terminology

The following terminology is used in the rest of the document.

- A **program** is a single .patch file comprised of one or more changes.
- A **change** is a single transformation in a patch file beginning with an
  `@`-header.
- **Metavariables** are declarations made inside the `@@` section (the
  metavariables section) of a change.
- The **patch** of a change is the actual transformation expressed as a unified
  diff.

# Parsing

A patch file is parsed in multiple stages. These stages are documented below.

## Sectioning

The first stage of parsing a patch file is to break it apart into independent
sections, each of which has its own language. We don't attempt to parse the
actual contents of the different sections at this stage.

For the purposes of sectioning, the grammar is the following.

```
# A .patch file (referred to as a program) consists of one or more changes.
program = change+;

# Each change has a header, metavariables section, and a patch.
change = header meta patch;

# If the change has a name, it is specified in the header.
header = "@@" | "@" change_name "@";
change_name = ident;

# The meta and patch sections are arbitrary blobs of bytes during sectioning.
meta = ???;
patch = ???;
```

Once the source is broken apart into sections, we parse each section separately
into the actual patch AST.

## Metavariables

The metavariables section contains zero or more `var` declarations in standard
Go style.

```
meta = var_decl*;
var_decl = "var" var_name ("," var_name)* type_name eol;
var_name = ident;
type_name = ident;
eol = '\n' | ';';
```

Using Go syntax here allows using the `go/scanner` package to parse this
section.

## Patch

Patches are specified as unified diffs of Go-ish syntax.

```diff
-x, err := foo(...)
+x, err := bar(...)
 if err != nil {
   ...
-  return err
+  return nil, err
 }
```

To parse a patch, we first break the unified diff apart into the two versions:
before and after. The above becomes,

```
Before                  After
------------------      ------------------
x, err := foo(...)      x, err := bar(...)
if err != nil {         if err != nil {
  ...                     ...
  return err              return nil, err
}                       }
```

Then each version is parsed separately (more on that later). Separating the
unified diff like this has a few benefits:

- We don't have to write a custom parser to understand leading `-`/`+`s.
- Parsing each version separately guarantees that they are both valid syntax.
- The same file parsing logic can be used to parse both, the Before and After
  versions of the file.

This is similar to the approach employed by Coccinelle, mentioned under [Basic
transformations].

  [Basic transformations]: http://coccinelle.lip6.fr/docs/main_grammar005.html#sec10

## pgo

A before/after version of the patch is not necessarily valid Go syntax. It is a
superset of the Go syntax which we have dubbed pgo, as in Patch Go. We want to
to use `go/parser` to parse pgo. To make this possible, we need to process pgo
further before using `go/parser`.

Consider the before section from above,

```go
x, err := foo(...)
if err != nil {
  ...
  return err
}
```

This code, despite being invalid Go syntax, still contains valid Go tokens. To
parse pgo, we scan through it with `go/scanner` and transform it so that it's
parseable with `go/parser`. For example, here's how the block above gets
transformed.


```
                   | package _
                   | func _() {
x, err := foo(...) |   x, err := foo(dts)
if err != nil {    |   if err != nil {
  ...              |     dts
  return err       |     return err
}                  |   }
                   | }
```

It can then be parsed with `go/parser`. The relevant portion of the AST is
extracted and transformed, replacing nodes at positions we recorded during
augmentation.

For example, `foo(...)` gets changed to `foo(dts)`, which gets parsed into the
following AST.

```go
&ast.CallExpr{
  Fun: &ast.Ident{Name: "foo"},
  Args: []ast.Expr{&ast.Ident{Name: "dts"}},
}
```

We recorded the position we placed `dts` at so the corresponding node in the
AST can be replaced with a `pgo.Dots` node.

```go
&ast.CallExpr{
  Fun: &ast.Ident{Name: "foo"},
  Args: []ast.Expr{&pgo.Dots{}},
}
```

Some transformations add or remove characters to the provided source. This
will affect the positions of all the user-input that follows after the
transformation. To adjust for this, we generate `PosAdjustment`s which inform
our parsing logic how much positions should be adjusted. For example, if a
`package` clause wasn't specified, one is generated and a `PosAdjustment` is
returned that indicates that starting at offset x in the augmented source,
reduce all positions by y to get positions in the original source.

# Compilation

The parsed patch is compiled to a secondary representation with thorough type
checking and validation. This representation is used by the [Engine] to drive
the patching algorithm.

  [Engine]: #engine

# Engine

The runtime engine for gopatch drives the patching behavior of the tool. The
engine interprets the parsed patch and operates on user-supplied Go files based
on the patch.

The engine is comprised of the following primary abstractions:

- **Matcher**, compiled from the "minus" section of a patch reports whether it
  matches Go code.
- **Replacer**, compiled from the "plus" section of a patch builds the AST that
  should be placed in place of the matched code.

Given a patch,

```diff
-err := f()
-if err != nil {
+if err := f(); err != nil {
    ...
 }
```

The following code gets compiled into a **Matcher**.

```go
err := f()
if err != nil {
    ...
}
```

And the following code gets compiled into a **Replacer**.

```go
if err := f(); err != nil {
    ...
}
```

## Data Sharing

Matchers and Replacers share information using **Data** objects. Data is an
immutable key-value store similar to `context.Context` with support for
adding and looking up values.

Roughly, the flow of information is similar to the following,


     (input)                     Compile
    .go files            .------ .patch -----.
        |               /                     \
        |        +-----|-----------------------|------+
        |        |     |         ENGINE        |      |
        |        |     v                       v      |
        v        |+---------+             +----------+|
     ast.File -->|| Matcher +--> Data --> | Replacer +|--> ast.File
                 |+---------+             +----------+|      |
                 |                                    |      |
                 +------------------------------------+      v
                                                          .go files
                                                           (output)


## Metavariables

Given a metavariable declaration,

    @@
    var f expression
    @@

Each occurrence of `f` in the minus section of the patch is converted into a
`MetavarMatcher` and each occurrence in the plus section is converted into a
`MetavarReplacer`.

Consider,

```diff
@@
var f expression
@@
-f(42)
+f(100)
```

The Go AST representation of `f(42)` is,

```go
&ast.CallExpr{
  Fun: &ast.Ident{Name: "f"},
  Args: []ast.Expr{
    &ast.BasicLit{
      Kind: token.INT,
      Value: "42",
    },
  },
}
```

The Matcher for it will look similar to the following.

```go
CallExprMatcher{
  Fun: MetavarMatcher{Name: "f"},
  Args: []ExprMatcher{
    BasicLitMatcher{
      Kind: ValueMatcher{Value: token.INT},
      Value: ValueMatcher{Value: "42"},
    }
  },
}
```

(Except we're actually using reflection to build the `CallExpr` matcher because
we don't want to write matchers by hand for every AST node type, but you get
the point.)

When the `CallExpr` matcher comes across a `CallExpr`, it will invoke the
`MetavarMatcher` on whatever is inside the `Fun` field of the `CallExpr`, which
in turn will capture the contents and store them on `Data` object.

```go
d, ok = matcher.Fun.Match(callExpr.Fun, d)
```

Similarly, the Replacer for `f(100)` will look similar to the following.

```
CallExprReplacer{
  Fun: MetavarReplacer{Name: "f"},
  Args: []ExprReplacer{
    BasicLitReplacer{
      Kind: ValueReplacer{Value: token.INT},
      Value: ValueReplacer{Value: "100"},
    }
  },
}
```

The `CallExpr` replacer will invoke the `MetavarReplacer` with the Data it
received, which will then produce the previously captured expression.

# Appendix: Position Tracking

gopatch relies on `"go/token".Pos` for position tracking. The usage and
concepts of that package are not obvious so this section attempts to explain
it.

## What and Why

The top-level type is a `FileSet`, comprised of multiple `File`s. The length of
each `File` is known, and so are the offsets within that file at which newlines
occur. A `Pos` indexes into a `FileSet`, providing a cheap int64-based pointer
to positional data: file name, line number, and column number.

To expand on that, imagine you have 4 files of lengths 10, 15, 5, and 16 bytes.
Adding these files to a FileSet in that order, you get a FileSet that may be
visualized like so.

```
 File # 1          2               3     4
        +----------+---------------+-----+----------------+
        | 10 bytes | 15 bytes      | 5 b | 16 bytes       |
        +----------+---------------+-----+----------------+
    Pos 1          12              28    34               51
```

This allows mapping an integer in the range `[1, 51)` to a specific file and an
offset within that file. Some examples are,

```
+-----+------+--------+
| Pos | File | Offset |
+-----+------+--------+
| 5   | 1    | 4      |
| 6   | 1    | 5      |
| 12  | 2    | 0      |
| 30  | 3    | 2      |
| 40  | 4    | 6      |
+-----+------+--------+
```

As previously mentioned, each `File` knows the offsets within itself at which
newlines occur. This makes it possible to convert an offset within that file to
a line and column number.

For example, if the file contents are `"foo\nbar\n"`, we know that newlines
occur at offsets 3 and 7. This makes it easy to say offset 6 maps to line 2,
column 3.

Additionally, because `Pos` is an integer, it is also easy to do simple
arithmetic on it to move between offsets in a file.

## How

An empty `FileSet` is created with `token.NewFileSet`.

```go
fset := token.NewFileSet()
```

Files can be added to a `FileSet` with `AddFile(name, base, size)` where `name`
is the name of the file (it does not have to be unique) and `size` is the
length of the file. `base` is the position within the `FileSet` at which the
range for this file starts. This will usually be `-1` to indicate that the
range for this file starts when the range for the previous file ends.

```
file1 := fset.AddFile("file1", -1, 10)  // base == 1
file2 := fset.AddFile("file2", -1, 15)  // base == 12
file3 := fset.AddFile("file3", -1, 5)   // base == 28
file4 := fset.AddFile("file4", -1, 16)  // base == 34

fmt.Println(fset.Base()) // base == 51
```

Once you have a `File`, it needs to be informed of the offsets at which new
lines start. This is done by using one of the following methods.

- Multiple `file.AddLine(offset)` calls informing it of the offsets at which
  new lines start. This is what `go/scanner` uses as it encounters newlines
  while tokenizing a Go file.
- A single `file.SetLines([]int)` call which accepts a series of offsets of the
  first characters of each line.
- A single `file.SetLinesForContent([]byte)` call which looks for newlines in
  the provided byte slice and picks up offsets accordingly.

Note that the offset is not for the newline character but for the character
immediately following that: The first character of each line.

Given a `File`, conversion between `Pos` and offset is possible with the
`Pos(int)` and `Offset(Pos)` methods.

Given a `File` or `FileSet`, complete positional information (file name, line
number, and column number) can be obtained with the `Position(Pos)` method
which returns a `Position` struct.

## Correlating positions between files

`File`s support recording overrides for positional information based on an
offset using the `AddLineColumnInfo(offset, filename, line, column)` method.
For any offset in a file, you can use `AddLineColumnInfo` to state a different
file name, line number, and column number that the contents of that line
correlate to.

The purpose of this API is to support error messages for issues found in
generated code that correlate back to the actual source file. This is how the
compiler supports `//line` directives.
