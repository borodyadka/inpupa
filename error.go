// Package inpupa contains the runtime contract used by code generated with the inpupa generator:
// typed errors of the FromRequest method.
package inpupa

import (
	"errors"
	"fmt"
)

// Kind — a class of an error returned by FromRequest.
type Kind int

const (
	// KindScalar — scalar parse error from header/query/cookie/path.
	KindScalar Kind = iota + 1
	// KindBody — request body decoding error (body[json]).
	KindBody
)

// Error — a typed error returned by the generated FromRequest method.
// Every error from FromRequest is either *Error itself or an envelope reachable via errors.As.
type Error struct {
	Kind   Kind
	Struct string // name of the input struct
	Field  string // Go field name
	Source string // "header" | "query" | "cookie" | "path" | "body[json]"
	Name   string // wire name; "" for body
	Value  string // raw value; "" for body
	Err    error  // primary cause (strconv, encoding/json, ...)
}

var _ error = (*Error)(nil)

// Error formats the message with the error location and the primary cause.
func (e *Error) Error() string {
	where := e.Struct + "." + e.Field
	var loc string
	switch {
	case e.Kind == KindBody:
		loc = e.Source
	case e.Name != "":
		loc = fmt.Sprintf("%s %q", e.Source, e.Name)
	default:
		loc = e.Source
	}
	if e.Value != "" {
		return fmt.Sprintf("inpupa: %s: parse value %q for %s: %v", where, e.Value, loc, e.Err)
	}
	return fmt.Sprintf("inpupa: %s: %s: %v", where, loc, e.Err)
}

// Unwrap returns the primary cause of the error.
func (e *Error) Unwrap() error { return e.Err }

// ErrEmptyBody — empty request body for a non-pointer body field.
var ErrEmptyBody = errors.New("empty request body")
