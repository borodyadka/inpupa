package genericstruct

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	inpupa "github.com/borodyadka/inpupa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reqWith(url string, set ...func(*http.Request)) *http.Request {
	r := httptest.NewRequest("GET", url, nil)
	for _, f := range set {
		f(r)
	}
	return r
}

func TestWrap_Full(t *testing.T) {
	r := reqWith("http://example.test/?id=7", func(r *http.Request) { r.Header.Set("X-Wrap", "w-1") })

	var in Wrap[string] // a concrete instantiation of the generic input
	require.NoError(t, in.FromRequest(r))
	assert.Equal(t, 7, in.ID)
	assert.Equal(t, "w-1", in.Origin)
	// Item is unannotated: never parsed, it stays at the zero value.
	assert.Empty(t, in.Item)
}

func TestWrap_AbsentStaysZero(t *testing.T) {
	r := reqWith("http://example.test/")

	var in Wrap[string]
	require.NoError(t, in.FromRequest(r))
	assert.Zero(t, in.ID)
	assert.Empty(t, in.Origin)
	assert.Empty(t, in.Item)
}

func TestWrap_ParseErrorTyped(t *testing.T) {
	r := reqWith("http://example.test/?id=notint")

	var in Wrap[string]
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie), "the error must be *inpupa.Error: %v", err)
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "Wrap", ie.Struct)
	assert.Equal(t, "ID", ie.Field)
	assert.Equal(t, "query", ie.Source)
	assert.Equal(t, "id", ie.Name)
	assert.Equal(t, "notint", ie.Value)
	require.NotNil(t, ie.Err)
}

func TestSlab_Full(t *testing.T) {
	r := reqWith("http://example.test/?label=l1")
	r.SetPathValue("ref", "9")

	var in Slab[Label] // the named string type satisfies the Word constraint
	require.NoError(t, in.FromRequest(r))
	assert.Equal(t, "l1", in.Label)
	assert.Equal(t, int64(9), in.Ref)
	// Piece is unannotated: never parsed.
	assert.Empty(t, in.Piece)
}

func TestSlab_ParseErrorTyped(t *testing.T) {
	r := reqWith("http://example.test/")
	r.SetPathValue("ref", "x")

	var in Slab[Label]
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "Slab", ie.Struct)
	assert.Equal(t, "Ref", ie.Field)
	assert.Equal(t, "path", ie.Source)
	assert.Equal(t, "ref", ie.Name)
	assert.Equal(t, "x", ie.Value)
	require.NotNil(t, ie.Err)
}
