// Package base64ops provides the pure encode/decode logic used by the
// MCP plugin. It deliberately has zero dependencies on the extism PDK or
// the wasip1 build target so it can be exercised with native `go test`.
package base64ops

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// Encode encodes raw using the standard or URL-safe alphabet.
//
// The result is always padded. Callers that need unpadded output should
// strip the trailing '=' themselves; we keep padded output as the
// default to round-trip with the most common decoders.
func Encode(raw string, urlSafe bool) string {
	enc := base64.StdEncoding
	if urlSafe {
		enc = base64.URLEncoding
	}
	return enc.EncodeToString([]byte(raw))
}

// Decode is permissive: it strips ASCII whitespace and then tries every
// common base64 variant (standard/URL-safe × padded/raw) in turn,
// returning the first successful decode. This matches what users
// usually mean by "decode this base64" and avoids forcing them to
// guess which alphabet or padding flavour was used.
//
// Returns an error only if none of the variants accept the input.
func Decode(s string) ([]byte, error) {
	cleaned := stripASCIIWhitespace(s)
	if cleaned == "" {
		return []byte{}, nil
	}
	encodings := [...]*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encodings {
		if b, err := enc.DecodeString(cleaned); err == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("input is not valid base64 (tried std, url-safe, padded, and raw alphabets)")
}

// stripASCIIWhitespace removes the four ASCII whitespace characters that
// commonly appear in base64 payloads from MIME, PEM, JWT debug dumps,
// or `base64` CLI output. Non-ASCII whitespace is left alone because
// it would never legitimately appear in base64 and is more useful as a
// decode error signal than silently dropped.
func stripASCIIWhitespace(s string) string {
	if !strings.ContainsAny(s, " \t\r\n") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
