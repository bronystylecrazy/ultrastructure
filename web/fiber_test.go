package web

import "testing"

func TestParseBodyLimit(t *testing.T) {
	tests := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{in: "4MB", want: 4_000_000},
		{in: "4MiB", want: 4 * 1024 * 1024},
		{in: "512KB", want: 512_000},
		{in: "512KiB", want: 512 * 1024},
		{in: "1GB", want: 1_000_000_000},
		{in: "4194304", want: 4194304},
		{in: "1.5MB", want: 1_500_000},
		{in: "12XB", wantErr: true},
		{in: "abc", wantErr: true},
	}

	for _, tc := range tests {
		got, err := ParseBodyLimit(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseBodyLimit(%q): expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseBodyLimit(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseBodyLimit(%q): got=%d want=%d", tc.in, got, tc.want)
		}
	}
}
