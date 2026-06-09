package github

import (
	"regexp"
	"testing"
)

func TestExtractWorkspace(t *testing.T) {
	pattern := regexp.MustCompile(`terraform apply \(([^,)]+)`)

	tests := []struct {
		job  string
		want string
	}{
		{"terraform apply (prod-gke)", "prod-gke"},
		{"terraform apply (prod-gke, asia-northeast1)", "prod-gke"},
		{"terraform apply (prod-network, us-central1)", "prod-network"},
	}

	for _, tt := range tests {
		got, err := extractWorkspace(tt.job, pattern)
		if err != nil {
			t.Fatalf("%q: %v", tt.job, err)
		}
		if got != tt.want {
			t.Fatalf("%q: got %q want %q", tt.job, got, tt.want)
		}
	}
}

func TestJobNameMatches(t *testing.T) {
	prefixes := []string{"terraform apply"}
	if !jobNameMatches("terraform apply (prod-gke)", prefixes) {
		t.Fatal("expected prefix match")
	}
	if jobNameMatches("terraform plan", prefixes) {
		t.Fatal("expected no match")
	}
}

func TestWorkflowMatches(t *testing.T) {
	patterns := []string{"terraform-apply.yml"}
	if !workflowMatches(".github/workflows/terraform-apply.yml", patterns) {
		t.Fatal("expected workflow path match")
	}
	if workflowMatches("ci.yml", patterns) {
		t.Fatal("expected no match")
	}
}
