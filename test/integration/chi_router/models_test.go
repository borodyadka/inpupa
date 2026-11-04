package chirouter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	inpupa "github.com/borodyadka/inpupa"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withChiParams — installs a chi route context carrying the given params. Uses only exported chi API.
func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rc := chi.NewRouteContext()
	for k, v := range params {
		rc.URLParams.Add(k, v)
	}
	ctx := r.Context()
	return r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rc))
}

func chiReq(url string, params map[string]string) *http.Request {
	return withChiParams(httptest.NewRequest("GET", url, nil), params)
}

func jsonReq(url, body string, params map[string]string) *http.Request {
	return withChiParams(httptest.NewRequest("POST", url, strings.NewReader(body)), params)
}

func TestAllSourcesInput_Full(t *testing.T) {
	r := chiReq("http://example.test/?category=7", map[string]string{"id": "p-42", "seq": "7", "limit": "42"})
	r.Header.Set("X-Request-ID", "r-1")
	r.AddCookie(&http.Cookie{Name: "session", Value: "s-1"})

	var in AllSourcesInput
	require.NoError(t, in.FromRequest(r))
	assert.Equal(t, "p-42", in.ID)
	assert.Equal(t, 7, in.Seq)
	require.True(t, in.Limit.IsSome()) // a present path value is wrapped in mo.Some
	assert.Equal(t, int64(42), in.Limit.MustGet())
	assert.Equal(t, "r-1", in.RequestID)
	assert.Equal(t, 7, in.CategoryID)
	assert.Equal(t, "s-1", in.Session)
}

func TestAllSourcesInput_AbsentPathIsZeroValues(t *testing.T) {
	r := chiReq("http://example.test/", nil) // a route context with no params: empty = absent

	var in AllSourcesInput
	require.NoError(t, in.FromRequest(r)) // zero values without an error
	assert.Equal(t, "", in.ID)
	assert.Equal(t, 0, in.Seq)
	assert.True(t, in.Limit.IsNone())
}

func TestAllSourcesInput_InvalidSeqPathTyped(t *testing.T) {
	r := chiReq("http://example.test/", map[string]string{"id": "p-1", "seq": "notint"})

	var in AllSourcesInput
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie), "the error must be *inpupa.Error: %v", err)
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "AllSourcesInput", ie.Struct)
	assert.Equal(t, "Seq", ie.Field)
	assert.Equal(t, "path", ie.Source)
	assert.Equal(t, "seq", ie.Name)
	assert.Equal(t, "notint", ie.Value)
	require.NotNil(t, ie.Err)

	// Fields before the failing one are already processed (declaration order).
	assert.Equal(t, "p-1", in.ID)
}

func TestAllSourcesInput_InvalidLimitPathTyped(t *testing.T) {
	r := chiReq("http://example.test/", map[string]string{"limit": "notint"})

	var in AllSourcesInput
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "AllSourcesInput", ie.Struct)
	assert.Equal(t, "Limit", ie.Field)
	assert.Equal(t, "path", ie.Source)
	assert.Equal(t, "limit", ie.Name)
	assert.Equal(t, "notint", ie.Value)
	require.NotNil(t, ie.Err)
}

func TestPointerBodyInput_Full(t *testing.T) {
	r := jsonReq("http://example.test/", `{"title":"t"}`, map[string]string{"ref": "r-1"})

	var in PointerBodyInput
	require.NoError(t, in.FromRequest(r))
	require.NotNil(t, in.Ref)
	assert.Equal(t, "r-1", *in.Ref)
	assert.Equal(t, "t", in.Payload.Title) // the inline body struct decodes via json tags
}

func TestPointerBodyInput_AbsentRefIsNil(t *testing.T) {
	r := jsonReq("http://example.test/", `{"title":"t"}`, nil)

	var in PointerBodyInput
	require.NoError(t, in.FromRequest(r)) // an absent pointer path param stays nil without an error
	assert.Nil(t, in.Ref)
}

func TestPointerBodyInput_EmptyBodyTyped(t *testing.T) {
	r := jsonReq("http://example.test/", "", map[string]string{"ref": "r-1"})

	var in PointerBodyInput
	err := in.FromRequest(r) // non-pointer body + empty body → ErrEmptyBody
	require.Error(t, err)
	assert.True(t, errors.Is(err, inpupa.ErrEmptyBody), "errors.Is(err, ErrEmptyBody): %v", err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindBody, ie.Kind)
	assert.Equal(t, "PointerBodyInput", ie.Struct)
	assert.Equal(t, "Payload", ie.Field)
	assert.Equal(t, "body[json]", ie.Source)

	// Fields before the body are already processed (declaration order).
	require.NotNil(t, in.Ref)
	assert.Equal(t, "r-1", *in.Ref)
}

// TestRealRouter_ParamsAndWildcard — extraction through an actual chi routing lifecycle: a {param}
// route and the bare trailing * catch-all served by chi.NewRouter. The handler calls FromRequest on
// the fixture struct and answers with both URLParam values plus the decoded tail, so one assertion
// covers the spec's pinned expectations for GET /users/u1/files/a/b/c.
func TestRealRouter_ParamsAndWildcard(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/users/{userID}/files/*", func(w http.ResponseWriter, req *http.Request) {
		var in UserFilesInput
		if err := in.FromRequest(req); err != nil {
			http.Error(w, "FromRequest: "+err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s|%s|%s", chi.URLParam(req, "userID"), chi.URLParam(req, "*"), in.Tail)
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/users/u1/files/a/b/c")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	// chi.URLParam "userID" == "u1", chi.URLParam "*" == "a/b/c" (the multi-segment tail), and the
	// generated method decodes the same tail from the same key.
	assert.Equal(t, "u1|a/b/c|a/b/c", string(body))
}
