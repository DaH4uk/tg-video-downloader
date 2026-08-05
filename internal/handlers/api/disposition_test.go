package api

import (
	"net/url"
	"strings"
	"testing"
)

func TestContentDispositionASCIIName(t *testing.T) {
	got := contentDisposition("TikTok - Something.mp4")
	want := `attachment; filename="TikTok - Something.mp4"; filename*=UTF-8''TikTok%20-%20Something.mp4`
	if got != want {
		t.Fatalf("contentDisposition() =\n%s\nwant\n%s", got, want)
	}
}

func TestContentDispositionNonASCIIName(t *testing.T) {
	original := "Видео 😀.mp4"
	got := contentDisposition(original)

	if !strings.HasPrefix(got, `attachment; filename="`) {
		t.Fatalf("unexpected prefix: %s", got)
	}

	fallback := got[len(`attachment; filename="`):strings.Index(got, `"; filename*=`)]
	for _, r := range fallback {
		if r > 127 {
			t.Fatalf("ascii fallback contains non-ascii rune %q: %s", r, fallback)
		}
	}
	if !strings.HasSuffix(fallback, ".mp4") {
		t.Fatalf("ascii fallback lost the extension: %s", fallback)
	}

	const marker = "filename*=UTF-8''"
	encoded := got[strings.Index(got, marker)+len(marker):]
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		t.Fatalf("filename* is not valid percent-encoding: %v", err)
	}
	if decoded != original {
		t.Fatalf("filename* decoded to %q, want %q", decoded, original)
	}
}

func TestContentDispositionQuotesAreStripped(t *testing.T) {
	got := contentDisposition(`we"ird\name.mp4`)
	fallback := got[len(`attachment; filename="`):strings.Index(got, `"; filename*=`)]
	if strings.ContainsAny(fallback, `"\`) {
		t.Fatalf("ascii fallback must not contain quotes or backslashes: %s", fallback)
	}
}
