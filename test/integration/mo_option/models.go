// Package mooption — integration tests for mo.Option[E] field types.
package mooption

import (
	"time"

	"github.com/samber/mo"
)

// Name — a named type over string.
type Name string

// OptionScalarInput — scalar option fields across all scalar sources.
type OptionScalarInput struct {
	// @input:query category
	CategoryID mo.Option[int]
	// @input:query tag
	Tag mo.Option[Name]
	// @input:header X-Trace
	TraceID mo.Option[string]
	// @input:path id
	ID mo.Option[int64]
	// @input:cookie page
	Page mo.Option[int64]
}

// OptionListInput — options over slices, allowed for query only.
type OptionListInput struct {
	// @input:query ids
	IDs mo.Option[[]int]
	// @input:query names
	Names mo.Option[[]string]
}

// OptionTimeInput — a TextUnmarshaler element (time.Time via UnmarshalText).
type OptionTimeInput struct {
	// @input:query at
	At mo.Option[time.Time]
}

// ValueOptionBodyInput — a VALUE option body field; one body field per struct.
type ValueOptionBodyInput struct {
	// @input:body[json]
	Value mo.Option[int]
}

// PointerOptionBodyInput — a POINTER option body field (the body path skips classification) in a
// separate struct: one body field per struct.
type PointerOptionBodyInput struct {
	// @input:body[json]
	Value *mo.Option[int]
}

// EnvelopeInput — the body decodes into an object whose field is an option, surfacing the
// missing-key and explicit-null semantics of mo.Option as a decoded struct field.
type EnvelopeInput struct {
	// @input:body[json]
	Doc struct {
		Value mo.Option[int]
	}
}
