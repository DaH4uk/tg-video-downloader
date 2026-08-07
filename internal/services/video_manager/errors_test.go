package video_manager

import (
	"testing"

	"github.com/pkg/errors"
)

func TestClassifyRejection(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   error
	}{
		{
			name:   "max filesize",
			output: "[download] File is larger than max-filesize (91234567 bytes > 524288000 bytes). Aborting download",
			want:   ErrTooLarge,
		},
		{
			name:   "match filter",
			output: `[youtube] abc: does not pass filter (duration < 900), skipping ..`,
			want:   ErrFiltered,
		},
		{
			name:   "unrelated failure",
			output: "ERROR: [youtube] abc: Video unavailable",
			want:   nil,
		},
		{
			name:   "empty output",
			output: "",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyRejection(tt.output)
			if !errors.Is(got, tt.want) {
				t.Fatalf("classifyRejection() = %v, want %v", got, tt.want)
			}
		})
	}
}
