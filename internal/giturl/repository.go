package giturl

import (
	"net/url"
	"strings"
)

// RepositoryFromURL extracts the repository short name from a Git remote URL,
// matching GitHub collect's repository field (e.g. "trading-api", org omitted).
func RepositoryFromURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	if strings.HasPrefix(raw, "git@") {
		return repositoryFromSSH(raw)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "", false
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", false
	}

	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	if name == "" {
		return "", false
	}
	return name, true
}

func repositoryFromSSH(raw string) (string, bool) {
	// git@github.com:org/repo.git
	colon := strings.Index(raw, ":")
	if colon < 0 {
		return "", false
	}
	path := strings.TrimPrefix(raw[colon+1:], "/")
	path = strings.TrimSuffix(path, ".git")
	if path == "" {
		return "", false
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", false
	}
	name := parts[len(parts)-1]
	if name == "" {
		return "", false
	}
	return name, true
}
