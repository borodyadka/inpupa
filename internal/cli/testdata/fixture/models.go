// Package fixture — a tiny input tree for the CLI flag-validation tests.
package fixture

// PingInput — non-path sources only, so builtin and chi runs emit byte-identical output here.
type PingInput struct {
	// @input:query ping
	Ping string
}
