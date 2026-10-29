package syntax

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

func TestE1_MissingName(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E1", "MissingName", "A"), "E1 on MissingName.A")
	assert.True(t, res.HasErrors())
}

func TestE2_BodyForms(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E2", "BodyNoBrackets", "B"))
	assert.True(t, hasDiag(res.Diagnostics, "E2", "BodyExtraToken", "C"))
	assert.True(t, hasDiag(res.Diagnostics, "E2", "BodyBadFormat", "F"))
}

func TestE2_UnknownSource(t *testing.T) {
	res := runGen(t)
	assert.True(t, hasDiag(res.Diagnostics, "E2", "UnknownSource", "D"))
	assert.True(t, hasDiag(res.Diagnostics, "E2", "UnknownSource", "E"))
	// The form source carries a note that it is planned.
	var sawFormNote bool
	for _, d := range res.Diagnostics {
		if d.Code == "E2" && d.Struct == "UnknownSource" && d.Field == "D" {
			sawFormNote = strings.Contains(d.Message, "planned")
		}
	}
	assert.True(t, sawFormNote, "the message for form should note that it is planned")
}

func TestW1_OrphanAnnotation(t *testing.T) {
	res := runGen(t)
	var w1 []gen.Diagnostic
	for _, d := range res.Diagnostics {
		if d.Code == "W1" {
			w1 = append(w1, d)
		}
	}
	require.Len(t, w1, 1, "expected exactly one W1 diagnostic")
	assert.Equal(t, gen.SevWarning, w1[0].Severity)
	assert.Empty(t, w1[0].Struct, "W1 is not attached to any struct")
	assert.Contains(t, w1[0].Message, "ignored")
}

func TestNoOutputOnErrors(t *testing.T) {
	res := runGen(t)
	assert.Empty(t, res.Written, "files are not written when validation errors are present")
}
