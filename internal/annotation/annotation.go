// Package annotation implements the @input:* annotation grammar.
package annotation

import (
	"fmt"
	"strings"
)

// Source — the origin of a field value.
type Source string

const (
	SourceHeader   Source = "header"
	SourceQuery    Source = "query"
	SourceCookie   Source = "cookie"
	SourcePath     Source = "path"
	SourceBodyJSON Source = "body[json]"
)

// Prefix — the annotation marker inside a comment.
const Prefix = "@input:"

// Directive — a parsed annotation.
type Directive struct {
	Source Source
	Name   string // wire name; "" for body
	Raw    string // original directive line (without "//")
}

// IsBody reports whether the source is the request body.
func (d *Directive) IsBody() bool { return d.Source == SourceBodyJSON }

// Error — a parse error with a generation code.
type Error struct {
	Code      string // "E1" | "E2"
	Directive string // original line text
	Message   string
	Hint      string // example of the correct form
}

func (e *Error) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("@input: %s [example: %s]", e.Message, e.Hint)
	}
	return fmt.Sprintf("@input: %s", e.Message)
}

// IsAnnotation reports whether the line (without "//") is an inpupa annotation.
func IsAnnotation(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), Prefix)
}

// Parse parses an annotation line (the text after "//").
// The line must start with "@input:"; otherwise E2 for an unknown form.
func Parse(line string) (*Directive, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, Prefix) {
		return nil, &Error{Code: "E2", Directive: line, Message: fmt.Sprintf("unknown directive %q", line), Hint: "@input:<source> <name>"}
	}
	payload := strings.TrimSpace(line[len(Prefix):])
	tokens := strings.Fields(payload)
	if len(tokens) == 0 {
		return nil, &Error{Code: "E2", Directive: line, Message: "missing source", Hint: "@input:<source> <name>"}
	}

	sourceTok := tokens[0]
	var d *Directive
	var perr *Error
	if sourceTok == string(SourceBodyJSON) || strings.HasPrefix(sourceTok, "body") {
		d, perr = parseBody(tokens, line)
	} else {
		d, perr = parseScalar(sourceTok, tokens, line)
	}
	if perr != nil {
		return nil, perr
	}
	return d, nil
}

func parseScalar(sourceTok string, tokens []string, raw string) (*Directive, *Error) {
	switch Source(sourceTok) {
	case SourceHeader, SourceQuery, SourceCookie, SourcePath:
	default:
		hint := "allowed sources: header, query, cookie, path, body[json]"
		msg := fmt.Sprintf("unknown source %q", sourceTok)
		if sourceTok == "form" {
			msg += " (planned)"
		}
		return nil, &Error{Code: "E2", Directive: raw, Message: msg, Hint: hint}
	}
	if len(tokens) < 2 {
		return nil, &Error{
			Code:      "E1",
			Directive: raw,
			Message:   "missing wire name for a scalar source",
			Hint:      fmt.Sprintf("@input:%s <name>", sourceTok),
		}
	}
	if len(tokens) > 2 {
		return nil, &Error{
			Code:      "E2",
			Directive: raw,
			Message:   "extra tokens after the wire name",
			Hint:      fmt.Sprintf("@input:%s <name>", sourceTok),
		}
	}
	return &Directive{Source: Source(sourceTok), Name: tokens[1], Raw: raw}, nil
}

func parseBody(tokens []string, raw string) (*Directive, *Error) {
	tok := tokens[0]
	if tok == "body" {
		return nil, &Error{
			Code:      "E2",
			Directive: raw,
			Message:   "the body source requires a format in brackets",
			Hint:      "@input:body[json]",
		}
	}
	if !strings.HasPrefix(tok, "body[") || !strings.HasSuffix(tok, "]") {
		return nil, &Error{
			Code:      "E2",
			Directive: raw,
			Message:   fmt.Sprintf("malformed body source %q", tok),
			Hint:      "@input:body[json]",
		}
	}
	format := strings.TrimSuffix(strings.TrimPrefix(tok, "body["), "]")
	if format != "json" {
		return nil, &Error{
			Code:      "E2",
			Directive: raw,
			Message:   fmt.Sprintf("unknown format %q (v1 supports json only)", format),
			Hint:      "@input:body[json]",
		}
	}
	if len(tokens) > 1 {
		return nil, &Error{
			Code:      "E2",
			Directive: raw,
			Message:   "unexpected token after body[<format>]: no name is accepted",
			Hint:      "@input:body[json]",
		}
	}
	return &Directive{Source: SourceBodyJSON, Raw: raw}, nil
}
