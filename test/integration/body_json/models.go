// Package bodyjson — tests for the body[json] source.
package bodyjson

import "time"

// PostBody — a named type for a body field.
type PostBody struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// CreatePostInput — the README example: an inline struct as a body field.
type CreatePostInput struct {
	// @input:header User-ID
	UserID string
	// @input:query category
	CategoryID int
	// @input:body[json]
	Post struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
}

// PublishInput — a pointer-type body field (empty body → nil without an error)
// and declaration-order processing of fields.
type PublishInput struct {
	// @input:query id
	ID string
	// @input:body[json]
	Body *PostBody
	// @input:query published
	Published bool
}

// UpdateInput — a map body field (any type for encoding/json) and a scalar after the body
// in declaration order.
type UpdateInput struct {
	// @input:path id
	ID string
	// @input:body[json]
	Data map[string]any
	// @input:query at
	At time.Time
}
