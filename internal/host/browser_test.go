package host

import "testing"

func TestBrowserCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		goos    string
		want    string
		wantErr bool
	}{
		{name: "windows", goos: "windows", want: "rundll32.exe"},
		{name: "darwin", goos: "darwin", want: "open"},
		{name: "unsupported", goos: "plan9", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, _, err := browserCommand(tt.goos, "https://example.com")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
