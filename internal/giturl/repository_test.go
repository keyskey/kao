package giturl

import "testing"

func TestRepositoryFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
		ok   bool
	}{
		{"https://github.com/org/trading-api.git", "trading-api", true},
		{"https://github.com/org/trading-api", "trading-api", true},
		{"git@github.com:org/trading-api.git", "trading-api", true},
		{"git@github.com:keyskey/kao.git", "kao", true},
		{"", "", false},
		{"https://example.com/not-a-repo", "", false},
	}

	for _, tt := range tests {
		got, ok := RepositoryFromURL(tt.url)
		if ok != tt.ok || got != tt.want {
			t.Errorf("RepositoryFromURL(%q) = (%q, %v), want (%q, %v)", tt.url, got, ok, tt.want, tt.ok)
		}
	}
}
