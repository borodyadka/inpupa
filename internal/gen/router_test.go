package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseRouter — the allow-set and the pinned error text (spec section 4, X = user input).
func TestParseRouter(t *testing.T) {
	for _, in := range []string{"builtin", "chi"} {
		got, err := ParseRouter(in)
		require.NoError(t, err)
		assert.Equal(t, Router(in), got)
	}
	for _, in := range []string{"", "Builtin", "CHi", "gorilla", "echo", " chi"} {
		_, err := ParseRouter(in)
		require.Error(t, err)
		assert.Equal(t, `invalid value "`+in+`" for --router: allowed values are builtin, chi`, err.Error())
	}
}

// TestPathGuardAccessors — the backend parsing table (spec section 5.1). pathGuard does not touch
// curPkg, so a renderer built with nil is safe for accessor checks.
func TestPathGuardAccessors(t *testing.T) {
	cases := []struct {
		name   string
		router Router
		want   string
	}{
		{"builtin", RouterBuiltin, `req.PathValue("id")`},
		{"chi", RouterChi, `chi.URLParam(req, "id")`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newRenderer(nil, c.router)
			assert.Equal(t, c.want, r.pathGuard("id"))
		})
	}
}

// TestPathGuardImportRegistration — the chi backend registers its import exactly once per file
// (the map deduplicates across path fields); builtin registers nothing.
func TestPathGuardImportRegistration(t *testing.T) {
	r := newRenderer(nil, RouterChi)
	assert.Equal(t, "chi", r.importName(chiImportPath, "chi"))
	r.pathGuard("a")
	r.pathGuard("b")
	assert.Len(t, r.imports, 1)
	assert.Equal(t, map[string]string{chiImportPath: "chi"}, r.imports)

	b := newRenderer(nil, RouterBuiltin)
	_ = b.pathGuard("id")
	assert.Empty(t, b.imports)
}
