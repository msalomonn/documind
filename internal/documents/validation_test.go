package documents

import (
	"strings"
	"testing"
)

func TestTextValidation(t *testing.T) {
	for _, tc := range []struct{ name, mime, body, code string }{
		{"note.txt", "text/plain", "Synthetic invoice\nTotal: 42", ""},
		{"empty.txt", "text/plain", "", "invalid_text"},
		{"fake.txt", "text/plain", "%PDF-1.7\n", "pdf_not_enabled"},
		{"scan.pdf", "application/pdf", "%PDF-1.7", "pdf_not_enabled"},
		{"binary.txt", "text/plain", "a\x00b", "invalid_text"},
		{"invalid.txt", "text/plain", "\xff", "invalid_text"},
		{"huge.txt", "text/plain", strings.Repeat("a", 200001), "text_too_long"},
		{"x.html", "text/html", "hello", "unsupported_file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.name, tc.mime, []byte(tc.body)); got != tc.code {
				t.Fatalf("got %q want %q", got, tc.code)
			}
		})
	}
}
