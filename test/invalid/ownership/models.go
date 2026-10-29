// Package ownership — ownership problems for fields and methods (E7–E10).
package ownership

import "net/http"

type UnexportedField struct {
	// @input:query hidden
	hidden string // E7: unexported field
}

type EmbeddedBase struct {
	Value string `json:"value"`
}

type AnnotatedEmbedded struct {
	// @input:query base
	EmbeddedBase // E8: annotation over an embedded field
}

type GenericInput[T any] struct {
	// @input:query v
	V T // E9: generic struct as input
}

type GenericPointerParam[T any] struct {
	// @input:query v
	V *T // E9: pointer to a type parameter — the recursive reference walk must catch it, not E4
}

type UserDefinedMethod struct {
	// @input:query x
	X string // valid field, but the method already exists → E10
}

func (m *UserDefinedMethod) FromRequest(req *http.Request) error {
	m.X = req.URL.Query().Get("x")
	return nil
}

type GenericUserMethod[T any] struct {
	// @input:query x
	X string // valid field, but the method already exists → E10
}

func (m *GenericUserMethod[T]) FromRequest(req *http.Request) error {
	m.X = req.URL.Query().Get("x")
	return nil
}
