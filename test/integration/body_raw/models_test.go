package bodyraw

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trackingBody wraps a reader and records whether Close was called.
type trackingBody struct {
	inner  io.ReadCloser
	closed bool
}

func (t *trackingBody) Read(p []byte) (int, error) { return t.inner.Read(p) }

func (t *trackingBody) Close() error {
	t.closed = true
	return t.inner.Close()
}

// reqWith wraps content into a tracking body and returns the request with it.
func reqWith(url, content string) (*http.Request, *trackingBody) {
	b := &trackingBody{inner: io.NopCloser(strings.NewReader(content))}
	r := httptest.NewRequest("POST", url, b)
	return r, b
}

func TestUploadInput_StreamPassthrough(t *testing.T) {
	const payload = `{"not":"decoded by the generator"}`
	r, b := reqWith("http://example.test/upload", payload)

	var in UploadInput
	require.NoError(t, in.FromRequest(r))
	require.NotNil(t, in.Payload)
	data, err := io.ReadAll(in.Payload)
	require.NoError(t, err)
	assert.Equal(t, payload, string(data)) // the bytes arrive untouched
	assert.False(t, b.closed, "FromRequest must not close a transferred body")
}

func TestUploadInput_EmptyBodyStillTransferred(t *testing.T) {
	r, _ := reqWith("http://example.test/upload", "")

	var in UploadInput
	require.NoError(t, in.FromRequest(r)) // an empty stream is not an error
	data, err := io.ReadAll(in.Payload)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestUploadInput_HandlerClosesTheStream(t *testing.T) {
	r, b := reqWith("http://example.test/upload", "x")

	var in UploadInput
	require.NoError(t, in.FromRequest(r))
	require.NoError(t, in.Payload.Close()) // ownership: the caller closes the stream
	assert.True(t, b.closed)
}

func TestAliasInput_AliasOfReadCloser(t *testing.T) {
	const payload = "aliased"
	r, _ := reqWith("http://example.test/alias", payload)

	var in AliasInput
	require.NoError(t, in.FromRequest(r))
	data, err := io.ReadAll(in.Data)
	require.NoError(t, err)
	assert.Equal(t, payload, string(data))
}

func TestEmbeddedInput_UserInterface(t *testing.T) {
	const payload = "embedded"
	r, _ := reqWith("http://example.test/embedded", payload)

	var in EmbeddedInput
	require.NoError(t, in.FromRequest(r))
	data, err := io.ReadAll(in.Stream)
	require.NoError(t, err)
	assert.Equal(t, payload, string(data))
}

func TestMixedInput_ScalarsAroundStream(t *testing.T) {
	const payload = "stream-bytes"
	r, b := reqWith("http://example.test/mixed?kind=audio", payload)
	r.Header.Set("User-ID", "u-1")

	var in MixedInput
	require.NoError(t, in.FromRequest(r))
	assert.Equal(t, "audio", in.Kind)
	assert.Equal(t, "u-1", in.UserID)
	data, err := io.ReadAll(in.Payload)
	require.NoError(t, err)
	assert.Equal(t, payload, string(data))
	assert.False(t, b.closed)
}
