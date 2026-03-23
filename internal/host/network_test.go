package host

import "testing"

func TestCandidateBaseURLsSpecificHost(t *testing.T) {
	t.Parallel()

	got := CandidateBaseURLs("127.0.0.1:38495")
	if len(got) != 1 || got[0] != "http://127.0.0.1:38495" {
		t.Fatalf("unexpected urls: %#v", got)
	}
}
