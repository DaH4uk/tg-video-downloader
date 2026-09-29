package redact

import "testing"

func TestSecrets(t *testing.T) {
	const token = "123456789:AAH4k-9xYz_abcdefghijklmnopqrstuvw"

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "getUpdates error",
			in:   `Post "https://api.telegram.org/bot` + token + `/getUpdates": context deadline exceeded`,
			want: `Post "https://api.telegram.org/bot<redacted>/getUpdates": context deadline exceeded`,
		},
		{
			name: "file download URL",
			in:   "https://api.telegram.org/file/bot" + token + "/videos/file_1.mp4",
			want: "https://api.telegram.org/file/bot<redacted>/videos/file_1.mp4",
		},
		{
			name: "bare token",
			in:   "token=" + token,
			want: "token=<redacted>",
		},
		{
			name: "multiple tokens",
			in:   token + " " + token,
			want: "<redacted> <redacted>",
		},
		{
			name: "no token",
			in:   "Failed to get updates, retrying in 3 seconds...",
			want: "Failed to get updates, retrying in 3 seconds...",
		},
		{
			name: "short colon-separated values are kept",
			in:   "started at 12:30, bot id 42:abc",
			want: "started at 12:30, bot id 42:abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Secrets(tt.in); got != tt.want {
				t.Errorf("Secrets(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if got := string(SecretsBytes([]byte(tt.in))); got != tt.want {
				t.Errorf("SecretsBytes(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
