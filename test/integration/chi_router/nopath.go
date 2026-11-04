package chirouter

// TraceInput — non-path sources only; the companion gen file must carry no chi import.
type TraceInput struct {
	// @input:header X-Trace-ID
	TraceID string
	// @input:query limit
	Limit int64
}
