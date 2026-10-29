package textunmarshaler

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	inpupa "github.com/borodyadka/inpupa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeInput_Full(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.test/?at=2026-01-02T03:04:05Z&timeout=1500000000", nil)
	r.Header.Set("X-Expires", "2027-05-06T07:08:09Z")

	var in TimeInput
	require.NoError(t, in.FromRequest(r)) // time.Time via UnmarshalText
	assert.Equal(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), in.At)
	// In this Go toolchain time.Duration has no TextUnmarshaler → parsed as int64 (ns).
	assert.Equal(t, 1500*time.Millisecond, in.Timeout)
	require.NotNil(t, in.Expires)
	assert.Equal(t, time.Date(2027, 5, 6, 7, 8, 9, 0, time.UTC), *in.Expires)
}

func TestTimeInput_BadDurationTyped(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.test/?timeout=not-a-number", nil)

	var in TimeInput
	err := in.FromRequest(r)
	require.Error(t, err)
	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, inpupa.KindScalar, ie.Kind)
	assert.Equal(t, "Timeout", ie.Field)
	assert.Equal(t, "not-a-number", ie.Value)
	require.NotNil(t, ie.Err)
}

func TestTimeInput_AbsentPointerIsNil(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.test/?at=2026-01-02T03:04:05Z", nil)

	var in TimeInput
	require.NoError(t, in.FromRequest(r))
	assert.Nil(t, in.Expires) // absent → nil
	assert.False(t, in.At.IsZero())
}

const validUUID = "0123456789abcdef0123456789abcdef"

func TestUUIDInput_Full(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.test/?uuid="+validUUID+"&trace="+validUUID, nil)

	var in UUIDInput
	require.NoError(t, in.FromRequest(r))
	want := [16]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	assert.Equal(t, UUID{Bytes: want}, in.UUID) // custom TextUnmarshaler
	require.NotNil(t, in.Trace)
	assert.Equal(t, UUID{Bytes: want}, *in.Trace)
}

func TestUUIDInput_AbsentPointerIsNil(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.test/?uuid="+validUUID, nil)

	var in UUIDInput
	require.NoError(t, in.FromRequest(r))
	assert.Nil(t, in.Trace)
}

func TestUUIDInput_BadValueTypedAndUnwrap(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.test/?uuid=short", nil)

	var in UUIDInput
	err := in.FromRequest(r)
	require.Error(t, err)
	var ie *inpupa.Error
	require.True(t, errors.As(err, &ie))
	assert.Equal(t, "UUID", ie.Field)
	assert.Equal(t, "short", ie.Value)
	// The primary error comes from UnmarshalText (errors.Is through Unwrap).
	assert.True(t, errors.Is(err, errBadUUID))
}
