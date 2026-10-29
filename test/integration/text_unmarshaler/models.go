// Package textunmarshaler — types with UnmarshalText take precedence over the basic kind.
package textunmarshaler

import (
	"errors"
	"time"
)

// UUID — a user-defined type implementing encoding.TextUnmarshaler.
type UUID struct {
	Bytes [16]byte
}

// UnmarshalText parses a UUID from 32 hexadecimal characters (no dashes).
func (u *UUID) UnmarshalText(data []byte) error {
	if len(data) != 32 {
		return errBadUUID
	}
	for i := 0; i < 16; i++ {
		h, ok := hexNibble(data[2*i])
		if !ok {
			return errBadUUID
		}
		l, ok := hexNibble(data[2*i+1])
		if !ok {
			return errBadUUID
		}
		u.Bytes[i] = h<<4 | l
	}
	return nil
}

var errBadUUID = errors.New("textunmarshaler: invalid uuid")

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

// TimeInput — time.Time via UnmarshalText; time.Duration has no TextUnmarshaler in this
// toolchain and is parsed as int64 (nanoseconds).
type TimeInput struct {
	// @input:query at
	At time.Time
	// @input:query timeout
	Timeout time.Duration
	// @input:header X-Expires
	Expires *time.Time
}

// UUIDInput — a custom TextUnmarshaler and a pointer to it.
type UUIDInput struct {
	// @input:query uuid
	UUID UUID
	// @input:query trace
	Trace *UUID
}
