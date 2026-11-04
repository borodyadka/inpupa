package cli

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// run — Run with os.Stderr captured through a pipe; the CLI prints fatal errors on it (spec §4).
// Generation is scoped to the small committed fixture tree, never the repo root.
func run(t *testing.T, args ...string) (code int, stderr string) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	code = Run(args)
	w.Close()
	os.Stderr = old
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	return code, string(data)
}

// TestRun_InvalidRouterRejected — no generation runs; exit 2 with the pinned message (X = input).
func TestRun_InvalidRouterRejected(t *testing.T) {
	cases := []struct{ name, value string }{
		{"unknown", "gorilla"},
		{"empty", ""},
		{"case-sensitive", "Builtin"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, stderr := run(t, "--input", "testdata/fixture", "--router="+c.value)
			assert.Equal(t, 2, code)
			want := fmt.Sprintf("inpupa: invalid value %q for --router: allowed values are builtin, chi\n", c.value)
			assert.Equal(t, want, stderr)
		})
	}
}

// TestRun_ValidRouterValuesAccepted — the two allow-set values run generation on the fixture
// (byte-stable there: non-path sources only) and finish clean.
func TestRun_ValidRouterValuesAccepted(t *testing.T) {
	for _, value := range []string{"builtin", "chi"} {
		code, stderr := run(t, "--input", "testdata/fixture", "--router="+value)
		assert.Equal(t, 0, code, "router=%s", value)
		assert.Empty(t, stderr, "router=%s", value)
	}
}

// TestRun_DefaultRouterIsBuiltin — --router unset behaves as builtin on the same fixture.
func TestRun_DefaultRouterIsBuiltin(t *testing.T) {
	code, stderr := run(t, "--input", "testdata/fixture")
	assert.Equal(t, 0, code)
	assert.Empty(t, stderr)
}
