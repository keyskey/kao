package scope

import "testing"

func TestResolveIncludeExclude(t *testing.T) {
	include := []string{"trading-api", "sandbox-demo", "settlement-api"}
	exclude := []string{"sandbox-*"}

	got := Resolve(include, exclude, nil)
	want := []string{"trading-api", "settlement-api"}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestResolveFilterRepos(t *testing.T) {
	got := Resolve([]string{"a", "b", "c"}, nil, []string{"b"})
	if len(got) != 1 || got[0] != "b" {
		t.Fatalf("got %v, want [b]", got)
	}
}
