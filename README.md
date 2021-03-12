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
