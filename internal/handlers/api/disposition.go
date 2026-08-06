package api

import (
	"fmt"
	"strings"
)

// contentDisposition builds a header value that survives non-ASCII titles: an
// ASCII fallback for the filename parameter and an RFC 5987 form with the real
// name. Video titles routinely contain Cyrillic and emoji.
func contentDisposition(filename string) string {
	var ascii strings.Builder
	for _, r := range filename {
		if r < 128 && r != '"' && r != '\\' {
			ascii.WriteRune(r)
			continue
		}
		ascii.WriteByte('_')
	}

	fallback := ascii.String()
	if strings.Trim(fallback, "_") == "" {
		fallback = "video.mp4"
	}

	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", fallback, percentEncodeRFC5987(filename))
}

// percentEncodeRFC5987 encodes a string for use in RFC 5987 ext-value according to attr-char rules.
// attr-char includes: ALPHA / DIGIT / "!" / "#" / "$" / "&" / "+" / "-" / "." / "^" / "_" / "`" / "|" / "~"
// All other bytes (including non-ASCII UTF-8 sequences) are percent-encoded.
func percentEncodeRFC5987(s string) string {
	var result strings.Builder
	for _, b := range []byte(s) {
		// Check if byte is in the attr-char set
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') ||
			b == '!' || b == '#' || b == '$' || b == '&' || b == '+' || b == '-' ||
			b == '.' || b == '^' || b == '_' || b == '`' || b == '|' || b == '~' {
			result.WriteByte(b)
		} else {
			fmt.Fprintf(&result, "%%%02X", b)
		}
	}
	return result.String()
}
