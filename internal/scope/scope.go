package scope

import (
	"path/filepath"
	"strings"
)

// Resolve returns repository names matching include/exclude globs.
// When filterRepos is non-empty, only those names are considered (CLI --repository).
func Resolve(include, exclude, filterRepos []string) []string {
	candidates := include
	if len(filterRepos) > 0 {
		candidates = filterRepos
	}

	var result []string
	seen := make(map[string]bool)
	for _, name := range candidates {
		if seen[name] {
			continue
		}
		if excluded(name, exclude) {
			continue
		}
		if len(include) > 0 && len(filterRepos) == 0 && !matched(name, include) {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result
}

func excluded(name string, patterns []string) bool {
	for _, p := range patterns {
		if matched(name, []string{p}) {
			return true
		}
	}
	return false
}

func matched(name string, patterns []string) bool {
	for _, p := range patterns {
		ok, _ := filepath.Match(p, name)
		if ok {
			return true
		}
		// Also support prefix wildcards like sandbox-*
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(name, prefix) {
				return true
			}
		}
	}
	return false
}
