// Package syntax — intentionally broken annotation forms (E1/E2, W1).
// The package compiles as Go code; only inpupa views it as invalid.
package syntax

type MissingName struct {
	// @input:query
	A string // E1: no wire name
}

type BodyNoBrackets struct {
	// @input:body
	B string // E2: body without brackets
}

type BodyExtraToken struct {
	// @input:body[json] extra
	C string // E2: extra token after the format
}

type UnknownSource struct {
	// @input:form x
	D string // E2: form — planned

	// @input:querry y
	E string // E2: typo in the source
}

type BodyBadFormat struct {
	// @input:body[xml]
	F string // E2: unknown format
}

// W1: an annotation over a function belongs to no field.
//
// @input:query orphan
func Helper() {}
