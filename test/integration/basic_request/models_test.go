package basicrequest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func setCookie(name, value string) func(*http.Request) {
	return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: name, Value: value}) }
}

func TestListPostsInput_Full(t *testing.T) {
	r := reqWith("http://example.test/posts?category=7&limit=12&q=x&active=true&score=1.5&big=99&u8=200&port=8080&tag=n1",
		func(r *http.Request) { r.Header.Set("X-Request-ID", "r-1") },
		setCookie("session", "s-1"),
	)
	r.SetPathValue("id", "p-42")

	var in ListPostsInput
	require.NoError(t, in.FromRequest(r))
	assert.Equal(t, 7, in.CategoryID)
	assert.Equal(t, int32(12), in.Limit)
	assert.Equal(t, "x", in.Query)
	assert.Equal(t, "r-1", in.RequestID)
	assert.Equal(t, "p-42", in.ID)
	assert.Equal(t, "s-1", in.Session)
	assert.True(t, in.Active)
	assert.InDelta(t, 1.5, in.Score, 1e-9)
	assert.Equal(t, int64(99), in.Big)
	assert.Equal(t, uint8(200), in.U8)
	assert.Equal(t, Port(8080), in.Port)
	assert.Equal(t, Name("n1"), in.Tag)
}

func TestListPostsInput_EmptyValuesAreZero(t *testing.T) {
	r := reqWith("http://example.test/?category=&q=")

	var in ListPostsInput
	require.NoError(t, in.FromRequest(r)) // an empty value means the zero value without an error
	assert.Equal(t, 0, in.CategoryID)
	assert.Equal(t, "", in.Query)
	assert.False(t, in.Active)
}

func TestListPostsInput_ParseErrorTyped(t *testing.T) {
	r := reqWith("http://example.test/?category=7&limit=notint")

	var in ListPostsInput
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie), "the error must be *inpupa.Error: %v", err)
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "ListPostsInput", ie.Struct)
	assert.Equal(t, "Limit", ie.Field)
	assert.Equal(t, "query", ie.Source)
	assert.Equal(t, "limit", ie.Name)
	assert.Equal(t, "notint", ie.Value)
	require.NotNil(t, ie.Err)

	// Fields before the failing one are already processed (declaration order).
	assert.Equal(t, 7, in.CategoryID)
}

func TestListPostsInput_OverflowTyped(t *testing.T) {
	r := reqWith("http://example.test/?limit=99999999999999999999")

	var in ListPostsInput
	err := in.FromRequest(r)
	require.Error(t, err)
	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, "99999999999999999999", ie.Value)
	assert.IsType(t, (*strconv.NumError)(nil), ie.Err)
}

func TestFiltersInput_AbsentPointersAndSlicesAreNil(t *testing.T) {
	r := reqWith("http://example.test/?x=1")

	var in FiltersInput
	require.NoError(t, in.FromRequest(r)) // absent → nil
	assert.Nil(t, in.Note)
	assert.Nil(t, in.Max)
	assert.Nil(t, in.Tags)
	assert.Nil(t, in.Nums)
	assert.Nil(t, in.IDs)
}

func TestFiltersInput_Present(t *testing.T) {
	r := reqWith("http://example.test/?note=n&max=5&tags=a&tags=&tags=b&nums=1&nums=2&ids=3&ids=4")

	var in FiltersInput
	require.NoError(t, in.FromRequest(r))
	require.NotNil(t, in.Note)
	assert.Equal(t, "n", *in.Note)
	require.NotNil(t, in.Max)
	assert.Equal(t, 5, *in.Max)
	assert.Equal(t, []string{"a", "b"}, in.Tags) // empty elements are discarded
	assert.Equal(t, []int64{1, 2}, in.Nums)      // URL order is preserved
	require.NotNil(t, in.IDs)
	assert.Equal(t, []int{3, 4}, *in.IDs) // *[]E: present → non-nil
}

func TestFiltersInput_DuplicateScalarTakesFirstNonEmpty(t *testing.T) {
	r := reqWith("http://example.test/?max=1&max=&max=2")

	var in FiltersInput
	require.NoError(t, in.FromRequest(r))
	require.NotNil(t, in.Max)
	assert.Equal(t, 1, *in.Max) // the first non-empty value wins
}

func TestFiltersInput_SliceOnlyEmptyValuesIsNil(t *testing.T) {
	r := reqWith("http://example.test/?ids=&ids=")

	var in FiltersInput
	require.NoError(t, in.FromRequest(r))
	assert.Nil(t, in.IDs) // only empty values → nil
}
