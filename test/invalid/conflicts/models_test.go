package conflicts

import (
	"strings"
	"testing"

	"github.com/borodyadka/inpupa/internal/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGen(t *testing.T) *gen.Result {
	t.Helper()
	res, err := gen.Generate(gen.Options{InputDir: "."})
	require.NoError(t, err)
	return res
}

func hasDiag(ds []gen.Diagnostic, code, structure, field string) bool {
	for _, d := range ds {
		if d.Code == code && d.Struct == structure && d.Field == field {
			return true
		}
	}
	return false
}

func TestE3_TwoAnnotationsOnOneField(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E3", "TwoAnnotations", "A"))
}

func TestE5_NameClashInSameSource(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E5", "NameClash", "S2"))
	var sawFirst bool
	for _, d := range res.Diagnostics {
		if d.Code == "E5" && d.Struct == "NameClash" {
			sawFirst = strings.Contains(d.Message, "S1")
		}
	}
	assert.True(t, sawFirst, "the E5 message must mention the first field")
}

func TestE6_TwoBodies(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E6", "TwoBodies", "B2"))
}

func TestDifferentSourcesSameNameAllowed(t *testing.T) {
	res := runGen(t)
	for _, d := range res.Diagnostics {
		if d.Severity == gen.SevError {
			assert.NotEqual(t, "SameNameDifferentSources", d.Struct,
				"the same name in different sources is not an error: %s", d.Render())
		}
	}
}

func TestNoOutputOnErrors(t *testing.T) {
	res := runGen(t)
	assert.True(t, res.HasErrors())
	assert.Empty(t, res.Written)
}
