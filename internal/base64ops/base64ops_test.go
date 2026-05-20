package base64ops

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		urlSafe bool
		want    string
	}{
		{"empty", "", false, ""},
		{"std ascii", "hello", false, "aGVsbG8="},
		{"std utf8", "héllo", false, "aMOpbGxv"},
		{"std needs padding", "f", false, "Zg=="},
		{"std no padding needed", "foo", false, "Zm9v"},
		{"url-safe distinguishes from std", "??>>", true, "Pz8-Pg=="},
		{"std same string differs from url-safe", "??>>", false, "Pz8+Pg=="},
		{"binary high bytes", "\xff\xfe\xfd", false, "//79"},
		{"url-safe binary high bytes", "\xff\xfe\xfd", true, "__79"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Encode(tt.input, tt.urlSafe)
			if got != tt.want {
				t.Errorf("Encode(%q, urlSafe=%v) = %q, want %q", tt.input, tt.urlSafe, got, tt.want)
			}
		})
	}
}

func TestDecode_Successes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"std padded", "aGVsbG8=", "hello"},
		{"std no padding needed", "Zm9v", "foo"},
		{"std double padding", "Zg==", "f"},
		{"url-safe padded", "Pz8-Pg==", "??>>"},
		{"raw std (no padding)", "aGVsbG8", "hello"},
		{"raw url-safe (no padding)", "Pz8-Pg", "??>>"},
		{"with newlines (PEM-style)", "aGVsbG8gd29y\nbGQ=", "hello world"},
		{"with CRLF", "aGVsbG8gd29y\r\nbGQ=", "hello world"},
		{"with spaces and tabs", "aGVs bG8g\td29ybGQ=", "hello world"},
		{"std mid-string padding tolerated via fallback", "aGVsbG8=\n", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.input)
			if err != nil {
				t.Fatalf("Decode(%q) returned error: %v", tt.input, err)
			}
			if !bytes.Equal(got, []byte(tt.want)) {
				t.Errorf("Decode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecode_Failures(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"non base64 alphabet", "!!!not-base64!!!"},
		{"single dangling char (invalid length even after raw)", "a"},
		{"std+url-safe mixed alphabets", "Pz8+-g=="},
		{"non-ascii whitespace not stripped", "aGVsbG8=\u00a0"}, // NBSP
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(tt.input)
			if err == nil {
				t.Fatalf("Decode(%q) succeeded, want error", tt.input)
			}
			if !strings.Contains(err.Error(), "not valid base64") {
				t.Errorf("Decode(%q) error = %q, want it to contain %q",
					tt.input, err.Error(), "not valid base64")
			}
		})
	}
}

// TestRoundTrip ensures Encode → Decode returns the original bytes for
// both alphabets, covering the union of inputs the plugin is likely to
// see (text, UTF-8, full byte range).
func TestRoundTrip(t *testing.T) {
	inputs := []string{
		"",
		"hello",
		"héllo world",
		"\x00\x01\x02\xff\xfe\xfd",
		strings.Repeat("a", 1024), // exercise long input
	}
	for _, in := range inputs {
		for _, urlSafe := range []bool{false, true} {
			encoded := Encode(in, urlSafe)
			decoded, err := Decode(encoded)
			if err != nil {
				t.Errorf("round-trip Decode(Encode(%q, urlSafe=%v)=%q) failed: %v",
					in, urlSafe, encoded, err)
				continue
			}
			if string(decoded) != in {
				t.Errorf("round-trip mismatch: in=%q urlSafe=%v encoded=%q decoded=%q",
					in, urlSafe, encoded, decoded)
			}
		}
	}
}

func TestStripASCIIWhitespace(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"abc", "abc"},
		{"a b\tc\nd\re", "abcde"},
		{"   \t\n\r   ", ""},
		{"héllo\nworld", "hélloworld"}, // multibyte rune preserved
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := stripASCIIWhitespace(tt.in); got != tt.want {
				t.Errorf("stripASCIIWhitespace(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Benchmark to make sure the fast path (no whitespace) doesn't allocate.
func BenchmarkDecode_NoWhitespace(b *testing.B) {
	input := Encode(strings.Repeat("payload", 100), false)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Decode(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode_WithWhitespace(b *testing.B) {
	raw := Encode(strings.Repeat("payload", 100), false)
	// inject newlines like PEM
	var sb strings.Builder
	for i := 0; i < len(raw); i += 64 {
		end := i + 64
		if end > len(raw) {
			end = len(raw)
		}
		sb.WriteString(raw[i:end])
		sb.WriteByte('\n')
	}
	input := sb.String()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Decode(input); err != nil {
			b.Fatal(err)
		}
	}
}
