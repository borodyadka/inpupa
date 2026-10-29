package annotation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Valid(t *testing.T) {
	tests := []struct {
		line   string
		source Source
		name   string
	}{
		{"@input:header User-ID", SourceHeader, "User-ID"},
		{"@input:query category", SourceQuery, "category"},
		{"@input:cookie session", SourceCookie, "session"},
		{"@input:path id", SourcePath, "id"},
		{"@input:body[json]", SourceBodyJSON, ""},
		// The wire name is a single token without spaces.
		{"@input:query user-id.v2", SourceQuery, "user-id.v2"},
		{"@input:query a_b-c.d", SourceQuery, "a_b-c.d"},
		// Extra whitespace is allowed as a separator.
		{"@input:header   X-Req-ID  ", SourceHeader, "X-Req-ID"},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			d, err := Parse(tt.line)
			require.NoError(t, err)
			assert.Equal(t, tt.source, d.Source)
			assert.Equal(t, tt.name, d.Name)
			assert.Equal(t, strings.TrimSpace(tt.line), d.Raw)
		})
	}
}

func TestParse_Errors(t *testing.T) {
	tests := []struct {
		line    string
		code    string
		msgPart string
	}{
		// E1 — missing wire name for scalar sources.
		{"@input:header", "E1", "missing wire name"},
		{"@input:query", "E1", "missing wire name"},
		{"@input:cookie", "E1", "missing wire name"},
		{"@input:path", "E1", "missing wire name"},
		// E2 — unknown sources and forms.
		{"@input:", "E2", "missing source"},
		{"@input:form x", "E2", "planned"},
		{"@input:querry x", "E2", `unknown source "querry"`},
		{"@input:Body x", "E2", `unknown source "Body"`},
		{"@input:body", "E2", "requires a format in brackets"},
		{"@input:body[json] extra", "E2", "unexpected token after body[<format>]"},
		{"@input:body[xml]", "E2", `unknown format "xml"`},
		{"@input:body[json", "E2", "malformed body source"},
		{"@input:query a b", "E2", "extra tokens after the wire name"},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			d, err := Parse(tt.line)
			require.Error(t, err)
			assert.Nil(t, d)
			var ae *Error
			require.ErrorAs(t, err, &ae)
			assert.Equal(t, tt.code, ae.Code)
			assert.Contains(t, ae.Message, tt.msgPart)
			assert.NotEmpty(t, ae.Hint, "the error should carry a hint")
		})
	}
}

func TestIsAnnotation(t *testing.T) {
	assert.True(t, IsAnnotation("@input:query x"))
	assert.True(t, IsAnnotation("   @input:header y"))
	assert.False(t, IsAnnotation("// @input:query x"))
	assert.False(t, IsAnnotation("@evl:validate required"))
	assert.False(t, IsAnnotation(""))
}
