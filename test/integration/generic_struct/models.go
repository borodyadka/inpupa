// Package genericstruct — integration tests for generic input structs: the generated
// receiver repeats the type parameter names only; annotated fields must be concrete.
package genericstruct

// Word — a named interface constraint over strings (a non-any constraint).
type Word interface {
	~string
}

// Label — a named type over string; the instantiation argument of Slab in the tests.
type Label string

// Wrap — a plain [T any] input: all annotated fields are concrete, the unannotated Item field is ignored.
type Wrap[T any] struct {
	// @input:query id
	ID int
	// @input:header X-Wrap
	Origin string
	Item   T
}

// Slab — a non-any constraint: the generated receiver must still repeat names only.
type Slab[T Word] struct {
	// @input:query label
	Label string
	// @input:path ref
	Ref   int64
	Piece T
}
