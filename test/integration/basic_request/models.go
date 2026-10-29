// Package basicrequest — behavioral tests for scalar sources.
package basicrequest

// Port — a named type over int.
type Port int

// Name — a named type over string.
type Name string

// ListPostsInput — all scalar sources and the supported kinds.
type ListPostsInput struct {
	// @input:query category
	CategoryID int
	// @input:query limit
	Limit int32
	// @input:query q
	Query string
	// @input:header X-Request-ID
	RequestID string
	// @input:path id
	ID string
	// @input:cookie session
	Session string
	// @input:query active
	Active bool
	// @input:query score
	Score float64
	// @input:query big
	Big int64
	// @input:query u8
	U8 uint8
	// @input:query port
	Port Port
	// @input:query tag
	Tag Name
}

// FiltersInput — pointer fields and multiple query values.
type FiltersInput struct {
	// @input:query note
	Note *string
	// @input:query max
	Max *int
	// @input:query tags
	Tags []string
	// @input:query nums
	Nums []int64
	// @input:query ids
	IDs *[]int
}
