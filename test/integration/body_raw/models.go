// Package bodyraw — tests for the bare body source (raw stream passthrough).
package bodyraw

import "io"

// Body — an alias over the raw stream type; aliases resolve transparently in the type rule.
type Body = io.ReadCloser

// Pipe — a user-defined interface embedding exactly the io.ReadCloser methods.
type Pipe interface {
	io.Reader
	io.Closer
}

// UploadInput — the basic form: the request stream lands in the field untouched.
type UploadInput struct {
	// @input:body
	Payload io.ReadCloser
}

// AliasInput — an alias of io.ReadCloser as the field type.
type AliasInput struct {
	// @input:body
	Data Body
}

// EmbeddedInput — a user interface with Read and Close via embedding.
type EmbeddedInput struct {
	// @input:body
	Stream Pipe
}

// MixedInput — scalar fields around a raw body; FromRequest must not close the stream.
type MixedInput struct {
	// @input:query kind
	Kind string
	// @input:header User-ID
	UserID string
	// @input:body
	Payload io.ReadCloser
}
