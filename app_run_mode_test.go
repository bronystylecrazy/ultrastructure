package us

import "testing"

// TestShouldRunServiceHostRunModes pins the run-mode decision that gates the
// kardianos service host. It complements TestShouldRunServiceHost with the
// flags-only and flags-before-token cases exercised by the lifecycle fix.
func TestShouldRunServiceHostRunModes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args runs host", args: nil, want: true},
		{name: "flags-only help runs host", args: []string{"--help"}, want: true},
		{name: "serve runs host", args: []string{"serve"}, want: true},
		{name: "service token skips host", args: []string{"service", "install"}, want: false},
		{name: "other command skips host", args: []string{"version"}, want: false},
		{name: "flags before token skips host", args: []string{"--verbose", "version"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRunServiceHost(tt.args, "service"); got != tt.want {
				t.Fatalf("shouldRunServiceHost(%v)=%v want=%v", tt.args, got, tt.want)
			}
		})
	}
}
