package gen

import "fmt"

// Router — backend for path parameter extraction (the --router flag).
// The zero value renders as builtin; the CLI layer is responsible for validation.
type Router string

const (
	RouterBuiltin Router = "builtin" // stdlib ServeMux: req.PathValue(name)
	RouterChi     Router = "chi"     // chi v5: chi.URLParam(req, name)
)

// chiImportPath — the chi package imported by generated files under --router=chi.
const chiImportPath = "github.com/go-chi/chi/v5"

// ParseRouter validates a --router value (case-sensitive; empty is invalid).
// The error text is pinned verbatim and printed as `inpupa: %v` by the CLI.
func ParseRouter(v string) (Router, error) {
	switch Router(v) {
	case RouterBuiltin:
		return RouterBuiltin, nil
	case RouterChi:
		return RouterChi, nil
	default:
		return "", fmt.Errorf("invalid value %q for --router: allowed values are builtin, chi", v)
	}
}

// pathGuard — the guard expression for a path wire name under the chosen backend.
// Any value other than RouterChi falls back to builtin (byte-identical to the
// pre-backend rendering); chi registers its import when a file emits at least one
// path field (the map deduplicates it).
func (r *renderer) pathGuard(name string) string {
	switch r.router {
	case RouterChi:
		return r.importName(chiImportPath, "chi") + ".URLParam(req, " + q(name) + ")"
	default:
		return "req.PathValue(" + q(name) + ")"
	}
}
