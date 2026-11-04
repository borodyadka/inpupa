package chirouter

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceInput_Full(t *testing.T) {
	r := chiReq("http://example.test/?limit=99", nil)
	r.Header.Set("X-Trace-ID", "t-1")

	var in TraceInput
	require.NoError(t, in.FromRequest(r))
	assert.Equal(t, "t-1", in.TraceID)
	assert.Equal(t, int64(99), in.Limit)
}

// TestNopathGenFileHasNoChiImport — import-absence rule (spec section 5.4): a file without path
// fields emits no chi import even under --router=chi.
func TestNopathGenFileHasNoChiImport(t *testing.T) {
	data, err := os.ReadFile("nopath.inpupa.gen.go")
	require.NoError(t, err)
	assert.NotContains(t, string(data), "github.com/go-chi/chi/v5")
}
