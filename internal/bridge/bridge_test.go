package bridge

import "testing"

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "http", input: "http://example.com"},
		{name: "https", input: "https://example.com/path?q=1"},
		{name: "trimmed", input: "  https://example.com  "},
		{name: "missing scheme", input: "example.com", wantErr: true},
		{name: "missing host", input: "https:///abc", wantErr: true},
		{name: "unsupported scheme", input: "file:///tmp/a.html", wantErr: true},
		{name: "javascript scheme", input: "javascript:alert(1)", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got url %q", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got == "" {
				t.Fatalf("expected normalized url")
			}
		})
	}
}
