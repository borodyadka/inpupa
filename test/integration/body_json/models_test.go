package bodyjson

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	inpupa "github.com/borodyadka/inpupa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jsonReq(url, body string) *http.Request {
	return httptest.NewRequest("POST", url, strings.NewReader(body))
}

func TestCreatePostInput_Full(t *testing.T) {
	r := jsonReq("http://example.test/posts?category=7", `{"title":"t","content":"c","unknown":1}`)
	r.Header.Set("User-ID", "u-9")

	var in CreatePostInput
	require.NoError(t, in.FromRequest(r)) // unknown JSON fields are ignored
	assert.Equal(t, "u-9", in.UserID)
	assert.Equal(t, 7, in.CategoryID)
	assert.Equal(t, "t", in.Post.Title)   // the inline body struct
	assert.Equal(t, "c", in.Post.Content) // json tags are honored
}

func TestCreatePostInput_EmptyBodyTyped(t *testing.T) {
	r := jsonReq("http://example.test/posts?category=7", "")
	r.Header.Set("User-ID", "u-9")

	var in CreatePostInput
	err := in.FromRequest(r)
	require.Error(t, err) // non-pointer body + empty body → error

	assert.True(t, errors.Is(err, inpupa.ErrEmptyBody), "errors.Is(err, ErrEmptyBody): %v", err)
	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindBody, ie.Kind)
	assert.Equal(t, "CreatePostInput", ie.Struct)
	assert.Equal(t, "Post", ie.Field)
	assert.Equal(t, "body[json]", ie.Source)

	// Fields before the body are processed.
	assert.Equal(t, "u-9", in.UserID)
	assert.Equal(t, 7, in.CategoryID)
}

func TestCreatePostInput_TruncatedJSON(t *testing.T) {
	r := jsonReq("http://example.test/posts", `{"title":`)

	var in CreatePostInput
	err := in.FromRequest(r)
	require.Error(t, err)
	assert.False(t, errors.Is(err, inpupa.ErrEmptyBody), "a truncated JSON is not an empty body")
	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindBody, ie.Kind)
	require.NotNil(t, ie.Err)
}

func TestPublishInput_PointerBodyEmptyIsNilNoError(t *testing.T) {
	r := jsonReq("http://example.test/publish?id=42&published=true", "")

	var in PublishInput
	require.NoError(t, in.FromRequest(r)) // pointer body → nil without an error
	assert.Nil(t, in.Body)
	// Scalars around the body are processed.
	assert.Equal(t, "42", in.ID)
	assert.True(t, in.Published)
}

func TestPublishInput_PointerBodyPresent(t *testing.T) {
	r := jsonReq("http://example.test/publish?id=42&published=true", `{"title":"t"}`)

	var in PublishInput
	require.NoError(t, in.FromRequest(r))
	require.NotNil(t, in.Body)
	assert.Equal(t, "t", in.Body.Title)
	assert.Equal(t, "42", in.ID)
	assert.True(t, in.Published)
}

func TestUpdateInput_MapBodyAndFieldsAfter(t *testing.T) {
	r := jsonReq("http://example.test/?at=2026-01-02T03:04:05Z", `{"a":1,"b":"x"}`)
	r.SetPathValue("id", "p-1")

	var in UpdateInput
	require.NoError(t, in.FromRequest(r))
	assert.Equal(t, "p-1", in.ID)
	require.NotNil(t, in.Data)
	assert.Equal(t, float64(1), in.Data["a"])
	assert.Equal(t, "x", in.Data["b"])
	assert.False(t, in.At.IsZero())
}
