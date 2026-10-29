package inpupa

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError_Messages(t *testing.T) {
	tests := []struct {
		name string
		e    *Error
		want string
	}{
		{
			name: "scalar with name and value",
			e: &Error{
				Kind: KindScalar, Struct: "CreatePostInput", Field: "CategoryID",
				Source: "query", Name: "category", Value: "abc",
				Err: errors.New("strconv: cannot parse"),
			},
			want: `inpupa: CreatePostInput.CategoryID: parse value "abc" for query "category": strconv: cannot parse`,
		},
		{
			name: "body with empty body sentinel",
			e: &Error{
				Kind: KindBody, Struct: "CreatePostInput", Field: "Post",
				Source: "body[json]", Err: ErrEmptyBody,
			},
			want: "inpupa: CreatePostInput.Post: body[json]: empty request body",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.e.Error())
		})
	}
}

func TestError_UnwrapAndIs(t *testing.T) {
	base := errors.New("strconv: cannot parse")
	e := &Error{Kind: KindScalar, Struct: "T", Field: "F", Source: "query", Name: "f", Value: "x", Err: base}

	assert.ErrorIs(t, e, base)

	wrapped := fmt.Errorf("handler: %w", e)
	var ie *Error
	assert.True(t, errors.As(wrapped, &ie))
	assert.Equal(t, "F", ie.Field)
	assert.ErrorIs(t, wrapped, base)
}

func TestError_EmptyBody_Is(t *testing.T) {
	// An empty body is caught via errors.Is(err, ErrEmptyBody) and as *inpupa.Error{Kind: KindBody} via errors.As.
	e := &Error{Kind: KindBody, Struct: "T", Field: "Post", Source: "body[json]", Err: ErrEmptyBody}

	var ie *Error
	assert.True(t, errors.As(e, &ie))
	assert.Equal(t, KindBody, ie.Kind)
	assert.ErrorIs(t, e, ErrEmptyBody)
}

func TestKind_Values(t *testing.T) {
	// iotas start at 1: constant values are stable as part of the API.
	assert.Equal(t, Kind(1), KindScalar)
	assert.Equal(t, Kind(2), KindBody)
}

func TestError_StrconvRoundtrip(t *testing.T) {
	// Verifies the exact construction the generator embeds in generated code:
	// the primary strconv error is carried in Err.
	raw := "99999999999999999999"
	_, err := strconv.ParseInt(raw, 10, 64)
	require.Error(t, err)
	e := &Error{Kind: KindScalar, Struct: "T", Field: "ID", Source: "query", Name: "id", Value: raw, Err: err}
	var ie *Error
	assert.True(t, errors.As(e, &ie))
	assert.Equal(t, raw, ie.Value)
	_, ok := e.Unwrap().(*strconv.NumError)
	assert.True(t, ok)
}
