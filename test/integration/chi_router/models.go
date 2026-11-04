// Package chirouter — integration tests for the --router=chi backend.
package chirouter

import "github.com/samber/mo"

// AllSourcesInput — all five sources at once: path (string, int, mo.Option[E]), header, query, cookie.
type AllSourcesInput struct {
	// @input:path id
	ID string
	// @input:path seq
	Seq int
	// @input:path limit
	Limit mo.Option[int64]
	// @input:header X-Request-ID
	RequestID string
	// @input:query category
	CategoryID int
	// @input:cookie session
	Session string
}

// PointerBodyInput — a *string path field (absent → nil) and an inline body[json] field.
type PointerBodyInput struct {
	// @input:path ref
	Ref *string
	// @input:body[json]
	Payload struct {
		Title string `json:"title"`
	}
}
