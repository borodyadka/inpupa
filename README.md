# Inpu(t)Pa(rsing) codegen

`inpupa` generates a `FromRequest(req *http.Request) error` method for input structs from `@input:*` annotations. The generated code is plain parsing without reflection (`strconv`, `encoding/json`, `UnmarshalText`) and runs in the build phase via `go generate`.

## Example

```go
type CreatePostInput struct {
    // @input:header User-ID
    UserID string
    // @input:query category
    CategoryID int
    // @input:body[json]
    Post struct {
        Title   string `json:"title"`
        Content string `json:"content"`
    }
}

func CreatePostHandler(res http.ResponseWriter, req *http.Request) {
    var input CreatePostInput
    if err := input.FromRequest(req); err != nil {
        // handle error
    }
    // work with input
}
```

The file with annotations wires in the generator:

```go
//go:generate inpupa --input .
```

## Usage

```sh
go install github.com/borodyadka/inpupa/cmd/inpupa@latest
go generate ./...
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--input` | `.` | directory to scan for annotated packages (recursively) |
| `--exclude` | — | comma-separated directories (relative to `--input`) skipped entirely; may be repeated |
| `--verbose` | off | verbose logging on stderr |
| `--werror` | off | treat warnings as errors (exit code 1) |

Exit codes: `0` — success; `1` — validation errors (or warnings with `--werror`); `2` — CLI/environment errors.

The generated file `<basename>.inpupa.gen.go` sits next to the source, is deterministic and rewritten only when its content changes.

## Sources

| Source | Annotation | Value from `*http.Request` |
| --- | --- | --- |
| `header` | `// @input:header User-ID` | `req.Header.Get` |
| `query` | `// @input:query category` | `req.URL.Query()`; slice fields and `mo.Option[[]E]` only here |
| `cookie` | `// @input:cookie session` | `req.Cookie` |
| `path` | `// @input:path id` | `req.PathValue` (stdlib `ServeMux` only, Go 1.22+) |
| `body[json]` | `// @input:body[json]` | `req.Body` via `encoding/json`; one body per struct |

Scalar source types: `string`, all integers, `float32/64`, `bool` (including named types over them), plus any type with `UnmarshalText` (`time.Time`, user-defined). `*T` — an optional value; `[]E`/`*[]E` — `query` only.

`mo.Option[E]` (module github.com/samber/mo) is accepted on `query`/`header`/`cookie`/`path` where scalars are, with `E` any supported scalar type; an absent or empty wire value leaves the zero Option (`None`), a present value is wrapped in `mo.Some`, and parse failures produce the same typed `*inpupa.Error` as scalars. `mo.Option[[]E]` mirrors the `[]E` rule (`query` only); `body[json]` fields of Option types decode via mo's own JSON implementation without special codegen. Using Option fields requires adding github.com/samber/mo to your module.

Input structs with type parameters are allowed when every annotated field has a concrete type: the generated method repeats the type parameter names in its receiver (constraints are never rendered); an annotated field referencing a type parameter is an error (E9), and a user-defined `FromRequest` on such a struct is still detected (E10).

Generation-time validation reports errors E1–E10 and warnings W1.

## Runtime errors

The module root runtime package (`github.com/borodyadka/inpupa`, package name `inpupa`) defines the typed errors: any error from `FromRequest` is reachable via `errors.As(err, &inpupa.Error{})`, an empty body additionally via `errors.Is(err, inpupa.ErrEmptyBody)`.

```go
var in CreatePostInput
if err := in.FromRequest(req); err != nil {
    var ie *inpupa.Error
    if errors.As(err, &ie) && ie.Field == "CategoryID" {
        // invalid query parameter category
    }
}
```

## Development

```sh
make build     # bin/inpupa
make test      # generation over test/ + go test ./...
make clean     # removes .inpupa.gen.go files under test/ and bin/
```

Tests: `test/integration/*` — behavioral httptest tests with committed `.inpupa.gen.go` files; `test/invalid/*` — intentionally broken annotations, diagnostic codes are asserted (excluded from generation via `--exclude invalid`).
