// Package conflicts — annotation conflicts (E3, E5, E6).
package conflicts

type TwoAnnotations struct {
	// @input:query a
	// @input:header b
	A string // E3: two annotations on one field
}

type NameClash struct {
	// @input:query sort
	S1 string

	// @input:query sort
	S2 int // E5: the same wire name in the same source
}

type TwoBodies struct {
	// @input:body[json]
	B1 map[string]string

	// @input:body[json]
	B2 string // E6: two body fields
}

// The same name in different sources is allowed (not an error).
type SameNameDifferentSources struct {
	// @input:query limit
	L1 int

	// @input:header limit
	L2 string
}
