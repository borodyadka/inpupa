// Package unsupported — unsupported types for scalar sources (E4).
package unsupported

import "github.com/samber/mo"

type BadTypes struct {
	// @input:header m
	M map[string]string // E4: map

	// @input:query c
	C chan int // E4: chan

	// @input:query f
	F func() // E4: func

	// @input:query u
	U uintptr // E4: uintptr
}

type SliceOnHeader struct {
	// @input:header tags
	T []string // E4: slice is allowed for query only
}

type BadSliceElem struct {
	// @input:query m
	M []map[string]string // E4: the slice element type is not supported
}

type BadOptionKinds struct {
	// @input:query o
	O mo.Option[*int] // E4: pointer elements are not allowed inside mo.Option[E]

	// @input:query r
	R mo.Result[int] // E4: generic types are not supported, except mo.Option[E]

	// @input:query p
	P *mo.Option[int] // E4: mo.Option[E] is not supported inside pointers or slices

	// @input:query s
	S []mo.Option[int] // E4: mo.Option[E] is not supported inside pointers or slices (slice element)
}
