package us

import "testing"

func TestShouldRunServiceHost(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: nil, want: true},
		{name: "serve command", args: []string{"serve"}, want: true},
		{name: "service command", args: []string{"service", "start"}, want: false},
		{name: "help command", args: []string{"help"}, want: false},
		{name: "version command", args: []string{"version"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRunServiceHost(tt.args, "service")
			if got != tt.want {
				t.Fatalf("shouldRunServiceHost(%v)=%v want=%v", tt.args, got, tt.want)
			}
		})
	}
}
