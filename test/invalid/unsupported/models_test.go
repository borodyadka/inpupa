package unsupported

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

func TestE4_UnsupportedScalarTypes(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E4", "BadTypes", "M"), "map")
	assert.True(t, hasDiag(res.Diagnostics, "E4", "BadTypes", "C"), "chan")
	assert.True(t, hasDiag(res.Diagnostics, "E4", "BadTypes", "F"), "func")
	assert.True(t, hasDiag(res.Diagnostics, "E4", "BadTypes", "U"), "uintptr")
}

func TestE4_MessageListsSupportedKinds(t *testing.T) {
	res := runGen(t)
	var saw bool
	for _, d := range res.Diagnostics {
		if d.Code == "E4" && d.Struct == "BadTypes" {
			saw = strings.Contains(d.Message, "supported")
			break
		}
	}
	assert.True(t, saw, "the E4 message must list the supported kinds")
}

func TestE4_SliceOnlyForQuery(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E4", "SliceOnHeader", "T"))
	var sawQuery bool
	for _, d := range res.Diagnostics {
		if d.Code == "E4" && d.Struct == "SliceOnHeader" {
			sawQuery = strings.Contains(d.Message, "query")
		}
	}
	assert.True(t, sawQuery, "the message should point to the query-only restriction")
}

func TestE4_BadSliceElement(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E4", "BadSliceElem", "M"))
}

func TestE4_BadOptionKinds(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E4", "BadOptionKinds", "O"), "pointer element inside an Option")
	assert.True(t, hasDiag(res.Diagnostics, "E4", "BadOptionKinds", "R"), "another mo generic instantiation")
	assert.True(t, hasDiag(res.Diagnostics, "E4", "BadOptionKinds", "P"), "Option behind a pointer at field position")
	assert.True(t, hasDiag(res.Diagnostics, "E4", "BadOptionKinds", "S"), "Option as a slice element")
}

func TestE4_BadOptionKindsMessages(t *testing.T) {
	res := runGen(t)
	wantMsgs := []struct{ field, msg string }{
		{"O", "type mo.Option[*int]: pointer elements are not allowed inside mo.Option[E]"},
		{"R", "type mo.Result[int]: generic types are not supported, except mo.Option[E]"},
		{"P", "type *mo.Option[int]: mo.Option[E] is not supported inside pointers or slices"},
		{"S", "type []mo.Option[int]: mo.Option[E] is not supported inside pointers or slices"},
	}
	for _, w := range wantMsgs {
		var saw bool
		for _, d := range res.Diagnostics {
			if d.Code == "E4" && d.Struct == "BadOptionKinds" && d.Field == w.field {
				saw = d.Message == w.msg
			}
		}
		assert.True(t, saw, "field %s: the E4 message must match verbatim", w.field)
	}
}

func TestNoOutputOnErrors(t *testing.T) {
	res := runGen(t)
	assert.Empty(t, res.Written)
}
