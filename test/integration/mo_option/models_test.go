package mooption

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func jsonReq(url, body string) *http.Request {
	return httptest.NewRequest("POST", url, strings.NewReader(body))
}

func TestOptionScalarInput_Full(t *testing.T) {
	r := reqWith("http://example.test/?category=7&tag=n1",
		func(r *http.Request) { r.Header.Set("X-Trace", "t-9") },
		setCookie("page", "3"),
	)
	r.SetPathValue("id", "42")

	var in OptionScalarInput
	require.NoError(t, in.FromRequest(r))
	require.True(t, in.CategoryID.IsSome())
	assert.Equal(t, 7, in.CategoryID.MustGet())
	require.True(t, in.Tag.IsSome()) // named element: converted from the raw value
	assert.Equal(t, Name("n1"), in.Tag.MustGet())
	require.True(t, in.TraceID.IsSome())
	assert.Equal(t, "t-9", in.TraceID.MustGet())
	require.True(t, in.ID.IsSome())
	assert.Equal(t, int64(42), in.ID.MustGet())
	require.True(t, in.Page.IsSome())
	assert.Equal(t, int64(3), in.Page.MustGet())
}

func TestOptionScalarInput_AbsentAreNone(t *testing.T) {
	r := reqWith("http://example.test/")

	var in OptionScalarInput
	require.NoError(t, in.FromRequest(r)) // an absent value stays at the zero value (None) without an error
	assert.True(t, in.CategoryID.IsNone())
	assert.True(t, in.Tag.IsNone())
	assert.True(t, in.TraceID.IsNone())
	assert.True(t, in.ID.IsNone())
	assert.True(t, in.Page.IsNone())
}

func TestOptionScalarInput_EmptyValuesAreNone(t *testing.T) {
	r := reqWith("http://example.test/?category=&tag=", setCookie("page", ""))

	var in OptionScalarInput
	require.NoError(t, in.FromRequest(r)) // an explicitly empty value also stays None
	assert.True(t, in.CategoryID.IsNone())
	assert.True(t, in.Tag.IsNone())
	assert.True(t, in.Page.IsNone())
}

func TestOptionScalarInput_DuplicateTakesFirstNonEmpty(t *testing.T) {
	r := reqWith("http://example.test/?category=&category=7")

	var in OptionScalarInput
	require.NoError(t, in.FromRequest(r))
	require.True(t, in.CategoryID.IsSome())
	assert.Equal(t, 7, in.CategoryID.MustGet()) // the first non-empty value wins
}

func TestOptionScalarInput_QueryParseErrorTyped(t *testing.T) {
	r := reqWith("http://example.test/?category=notint")

	var in OptionScalarInput
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie), "the error must be *inpupa.Error: %v", err)
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "OptionScalarInput", ie.Struct)
	assert.Equal(t, "CategoryID", ie.Field)
	assert.Equal(t, "query", ie.Source)
	assert.Equal(t, "category", ie.Name)
	assert.Equal(t, "notint", ie.Value)
	require.NotNil(t, ie.Err)
}

func TestOptionScalarInput_PathParseErrorTyped(t *testing.T) {
	r := reqWith("http://example.test/")
	r.SetPathValue("id", "x")

	var in OptionScalarInput
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "OptionScalarInput", ie.Struct)
	assert.Equal(t, "ID", ie.Field)
	assert.Equal(t, "path", ie.Source)
	assert.Equal(t, "id", ie.Name)
	assert.Equal(t, "x", ie.Value)
	require.NotNil(t, ie.Err)
}

func TestOptionScalarInput_CookieParseErrorTyped(t *testing.T) {
	r := reqWith("http://example.test/", setCookie("page", "x"))

	var in OptionScalarInput
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "OptionScalarInput", ie.Struct)
	assert.Equal(t, "Page", ie.Field)
	assert.Equal(t, "cookie", ie.Source)
	assert.Equal(t, "page", ie.Name)
	assert.Equal(t, "x", ie.Value)
	require.NotNil(t, ie.Err)
}

func TestOptionListInput_Present(t *testing.T) {
	r := reqWith("http://example.test/?ids=1&ids=&ids=2&names=a&names=&names=b")

	var in OptionListInput
	require.NoError(t, in.FromRequest(r))
	require.True(t, in.IDs.IsSome())               // at least one parsed → Some(list)
	assert.Equal(t, []int{1, 2}, in.IDs.MustGet()) // non-empty values in URL order, empties discarded
	require.True(t, in.Names.IsSome())
	assert.Equal(t, []string{"a", "b"}, in.Names.MustGet())
}

func TestOptionListInput_AbsentAreNone(t *testing.T) {
	r := reqWith("http://example.test/")

	var in OptionListInput
	require.NoError(t, in.FromRequest(r)) // absent → no assignment → None
	assert.True(t, in.IDs.IsNone())
	assert.True(t, in.Names.IsNone())
}

func TestOptionListInput_OnlyEmptyValuesAreNone(t *testing.T) {
	r := reqWith("http://example.test/?ids=&ids=")

	var in OptionListInput
	require.NoError(t, in.FromRequest(r)) // only empty values → None
	assert.True(t, in.IDs.IsNone())
}

func TestOptionListInput_ParseErrorTyped(t *testing.T) {
	r := reqWith("http://example.test/?ids=1&ids=abc")

	var in OptionListInput
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "OptionListInput", ie.Struct)
	assert.Equal(t, "IDs", ie.Field)
	assert.Equal(t, "query", ie.Source)
	assert.Equal(t, "ids", ie.Name)
	assert.Equal(t, "abc", ie.Value)
	require.NotNil(t, ie.Err)
}

func TestOptionTimeInput_Full(t *testing.T) {
	r := reqWith("http://example.test/?at=2026-01-02T03:04:05Z")

	var in OptionTimeInput
	require.NoError(t, in.FromRequest(r)) // TextUnmarshaler element wrapped into mo.Some
	require.True(t, in.At.IsSome())
	assert.Equal(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), in.At.MustGet())
}

func TestOptionTimeInput_AbsentIsNone(t *testing.T) {
	r := reqWith("http://example.test/")

	var in OptionTimeInput
	require.NoError(t, in.FromRequest(r))
	assert.True(t, in.At.IsNone())
}

func TestOptionTimeInput_BadValueTyped(t *testing.T) {
	r := reqWith("http://example.test/?at=bogus")

	var in OptionTimeInput
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "OptionTimeInput", ie.Struct)
	assert.Equal(t, "At", ie.Field)
	assert.Equal(t, "query", ie.Source)
	assert.Equal(t, "at", ie.Name)
	assert.Equal(t, "bogus", ie.Value)
	require.NotNil(t, ie.Err) // the UnmarshalText cause (time.ParseError) is preserved
}

func TestValueOptionBody_PresentIsSome(t *testing.T) {
	r := jsonReq("http://example.test/", "7")

	var in ValueOptionBodyInput
	require.NoError(t, in.FromRequest(r)) // delegation to mo's own UnmarshalJSON
	require.True(t, in.Value.IsSome())
	assert.Equal(t, 7, in.Value.MustGet())
}

func TestValueOptionBody_NullIsNone(t *testing.T) {
	r := jsonReq("http://example.test/", "null")

	var in ValueOptionBodyInput
	require.NoError(t, in.FromRequest(r)) // an explicit null → None
	assert.True(t, in.Value.IsNone())
}

func TestValueOptionBody_MalformedElementTyped(t *testing.T) {
	r := jsonReq("http://example.test/", `"abc"`)

	var in ValueOptionBodyInput
	err := in.FromRequest(r) // a string where the int element is expected
	require.Error(t, err)
	assert.False(t, errors.Is(err, inpupa.ErrEmptyBody), "a malformed element is not an empty body")

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindBody, ie.Kind)
	assert.Equal(t, "ValueOptionBodyInput", ie.Struct)
	assert.Equal(t, "Value", ie.Field)
	assert.Equal(t, "body[json]", ie.Source)
	require.NotNil(t, ie.Err)
}

func TestValueOptionBody_EmptyTyped(t *testing.T) {
	r := jsonReq("http://example.test/", "")

	var in ValueOptionBodyInput
	err := in.FromRequest(r) // a non-pointer body field + an empty body → ErrEmptyBody
	require.Error(t, err)
	assert.True(t, errors.Is(err, inpupa.ErrEmptyBody), "errors.Is(err, ErrEmptyBody): %v", err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindBody, ie.Kind)
	assert.Equal(t, "ValueOptionBodyInput", ie.Struct)
	assert.Equal(t, "Value", ie.Field)
}

func TestPointerOptionBody_EmptyIsNilNoError(t *testing.T) {
	r := jsonReq("http://example.test/", "")

	var in PointerOptionBodyInput
	require.NoError(t, in.FromRequest(r)) // a pointer body field + an empty body → nil without an error
	assert.Nil(t, in.Value)
}

func TestPointerOptionBody_NullIsNil(t *testing.T) {
	r := jsonReq("http://example.test/", "null")

	var in PointerOptionBodyInput
	require.NoError(t, in.FromRequest(r)) // an explicit null → the pointer stays nil
	assert.Nil(t, in.Value)
}

func TestPointerOptionBody_PresentIsSome(t *testing.T) {
	r := jsonReq("http://example.test/", "9")

	var in PointerOptionBodyInput
	require.NoError(t, in.FromRequest(r)) // json allocates the pointer and mo's impl fills Some
	require.NotNil(t, in.Value)
	require.True(t, in.Value.IsSome())
	assert.Equal(t, 9, in.Value.MustGet())
}

func TestPointerOptionBody_MalformedElementTyped(t *testing.T) {
	r := jsonReq("http://example.test/", `"abc"`)

	var in PointerOptionBodyInput
	err := in.FromRequest(r)
	require.Error(t, err)

	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindBody, ie.Kind)
	assert.Equal(t, "PointerOptionBodyInput", ie.Struct)
	assert.Equal(t, "Value", ie.Field)
	assert.Equal(t, "body[json]", ie.Source)
	require.NotNil(t, ie.Err)
}

func TestEnvelope_MissingFieldIsNone(t *testing.T) {
	r := jsonReq("http://example.test/", "{}")

	var in EnvelopeInput
	require.NoError(t, in.FromRequest(r)) // a missing key leaves the zero Option → None
	assert.True(t, in.Doc.Value.IsNone())
}

func TestEnvelope_NullFieldIsNone(t *testing.T) {
	r := jsonReq("http://example.test/", `{"value":null}`)

	var in EnvelopeInput
	require.NoError(t, in.FromRequest(r)) // an explicit null inside the object → None
	assert.True(t, in.Doc.Value.IsNone())
}

func TestEnvelope_PresentFieldIsSome(t *testing.T) {
	r := jsonReq("http://example.test/", `{"value":42}`)

	var in EnvelopeInput
	require.NoError(t, in.FromRequest(r))
	require.True(t, in.Doc.Value.IsSome())
	assert.Equal(t, 42, in.Doc.Value.MustGet())
}
