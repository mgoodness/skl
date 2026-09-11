package skl

import "testing"

func TestParseSource_GitHubShorthand(t *testing.T) {
	got := parseSource("owner/repo")
	if got.Kind != sourceKindGitHub {
		t.Fatalf("Kind = %v, want sourceKindGitHub", got.Kind)
	}
	if got.GitHub.Owner != "owner" || got.GitHub.Repo != "repo" {
		t.Errorf("GitHub = %+v, want Owner=%q Repo=%q", got.GitHub, "owner", "repo")
	}
}

func TestParseSource_GitHubFullURL(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"https, no trailing slash", "https://github.com/owner/repo"},
		{"https, trailing slash", "https://github.com/owner/repo/"},
		{"http", "http://github.com/owner/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSource(tt.src)
			if got.Kind != sourceKindGitHub {
				t.Fatalf("Kind = %v, want sourceKindGitHub", got.Kind)
			}
			if got.GitHub.Owner != "owner" || got.GitHub.Repo != "repo" {
				t.Errorf("GitHub = %+v, want Owner=%q Repo=%q", got.GitHub, "owner", "repo")
			}
		})
	}
}

func TestParseSource_LocalPaths(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"three path segments", "testdata/fixtures/simple-skill"},
		{"relative with leading dot", "./simple-skill"},
		{"absolute path", "/tmp/some-skill"},
		{"single segment, no slash", "just-a-name"},
		{"github.com URL with extra tree-path segments", "https://github.com/owner/repo/tree/main/skills/tdd"},
		{"shorthand-shaped but four segments (tree-path, out of scope)", "owner/repo/tree/main"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSource(tt.src)
			if got.Kind != sourceKindLocal {
				t.Errorf("Kind = %v, want sourceKindLocal for %q", got.Kind, tt.src)
			}
		})
	}
}

func TestGitHubSource_StringAndURL(t *testing.T) {
	src := GitHubSource{Owner: "owner", Repo: "repo"}
	if got := src.String(); got != "owner/repo" {
		t.Errorf("String() = %q, want %q", got, "owner/repo")
	}
	if got := src.URL(); got != "https://github.com/owner/repo" {
		t.Errorf("URL() = %q, want %q", got, "https://github.com/owner/repo")
	}
}
