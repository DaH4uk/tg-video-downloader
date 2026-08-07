package api

import (
	"testing"

	"github.com/pkg/errors"
)

func TestValidateVideoURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want error
	}{
		{"valid https url", "https://www.tiktok.com/@user/video/123", nil},
		{"valid public ip literal", "https://93.184.216.34/video.mp4", nil},
		{"http scheme is rejected", "http://example.com/video", ErrInvalidURL},
		{"missing scheme", "example.com/video", ErrInvalidURL},
		{"empty string", "", ErrInvalidURL},
		{"scheme without host", "https://", ErrInvalidURL},
		{"loopback ipv4", "https://127.0.0.1/video", ErrForbiddenHost},
		{"loopback ipv6", "https://[::1]/video", ErrForbiddenHost},
		{"private network", "https://192.168.1.10/video", ErrForbiddenHost},
		{"link local metadata", "https://169.254.169.254/latest/meta-data", ErrForbiddenHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateVideoURL(tt.raw)
			if !errors.Is(got, tt.want) {
				t.Fatalf("ValidateVideoURL(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
