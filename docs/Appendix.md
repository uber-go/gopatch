# Appendix

## Identifiers vs expressions vs statements

A simplified explanation of the difference between identifiers, expressions
and statements is,

- [**identifiers**] are names of things
- [**expressions**] are things that have values (you can pass these into
  functions), or refer to types
- [**statements**] are instructions to do things

  [**identifiers**]: https://golang.org/ref/spec#identifier
  [**expressions**]: https://golang.org/ref/spec#Expression
  [**statements**]: https://golang.org/ref/spec#Statement

Consider the following snippet.

```go
var bar Bar
if err := foo(bar.Baz()); err != nil {
  return err
}
```

It contains,

- identifiers: `err`, `foo`, `bar`, `Bar`, `Baz`

    ```
    var bar Bar
        '-' '-'
    if err := foo(bar.Baz()); err != nil {
       '-'    '-' '-' '-'     '-'
      return err
             '-'
    }
    ```

- expressions: `Bar`, `bar.Baz`, `bar.Baz()`, `foo(bar.Baz())`, `err`, `nil`,
  and `err != nil`

    ```
    var bar Bar
            '-'
                  .-------.
              .---|-------|.  .---------.
    if err := foo(bar.Baz()); err != nil {
                  '-'   |     '-'    '-'
                  '-----'
      return err
             '-'
    }
    ```

    Note that `bar` in `var bar Bar` is not an expression.

- statements: `var ...`, `err := ...`, `if ...`, and `return ...`

    ```
    var bar Bar
    '---------'
    if err := foo(bar.Baz()); err != nil {  -.
       '-------------------'                 |
      return err                             |
      '--------'                             |
    }                                       -'
    ```
