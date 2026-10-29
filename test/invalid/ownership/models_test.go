package ownership

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

func hasDiag(ds []gen.Diagnostic, code, structure string) bool {
	for _, d := range ds {
		if d.Code == code && d.Struct == structure {
			return true
		}
	}
	return false
}

func TestE7_UnexportedField(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E7", "UnexportedField"))
}

func TestE8_AnnotatedEmbedded(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E8", "AnnotatedEmbedded"))
}

func TestE9_GenericStruct(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E9", "GenericInput"))
}

func TestE9_PointerToTypeParam(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E9", "GenericPointerParam"))
	assert.False(t, hasDiag(res.Diagnostics, "E4", "GenericPointerParam"), "a pointer to a type parameter must be E9, not E4")
	var saw bool
	for _, d := range res.Diagnostics {
		if d.Code == "E9" && d.Struct == "GenericPointerParam" {
			saw = d.Field == "V" && d.Message == "field V uses type parameter T; parsing into unbound type parameters is not supported"
		}
	}
	assert.True(t, saw, "the field-level E9 must carry the dedicated type-parameter message")
}

func TestE10_UserDefinedFromRequest(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E10", "UserDefinedMethod"))
	var sawConflict bool
	for _, d := range res.Diagnostics {
		if d.Code == "E10" {
			sawConflict = true
		}
	}
	assert.True(t, sawConflict)
}

func TestE10_GenericReceiver(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E10", "GenericUserMethod"))
	var saw bool
	for _, d := range res.Diagnostics {
		if d.Code == "E10" && d.Struct == "GenericUserMethod" {
			saw = strings.HasSuffix(d.File, "models.go") && d.Line > 0 &&
				d.Message == "struct GenericUserMethod already has a user-defined FromRequest method in this package — conflicts with generation"
		}
	}
	assert.True(t, saw, "the E10 diagnostic must point at the user-defined generic receiver")
}

func TestNoOutputOnErrors(t *testing.T) {
	res := runGen(t)
	assert.Empty(t, res.Written)
}
