package skl

import (
	"strings"
	"testing"
)

func TestParseSource_GitHubShorthand(t *testing.T) {
	got, err := parseSource("owner/repo")
	if err != nil {
		t.Fatalf("parseSource() error = %v", err)
	}
	if got.Kind != sourceKindGitHub {
		t.Fatalf("Kind = %v, want sourceKindGitHub", got.Kind)
	}
	if got.GitHub.Owner != "owner" || got.GitHub.Repo != "repo" {
		t.Errorf("GitHub = %+v, want Owner=%q Repo=%q", got.GitHub, "owner", "repo")
	}
	if got.Ref != "" || got.Path != "" {
		t.Errorf("Ref/Path = %q/%q, want both empty for a plain shorthand source", got.Ref, got.Path)
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
			got, err := parseSource(tt.src)
			if err != nil {
				t.Fatalf("parseSource() error = %v", err)
			}
			if got.Kind != sourceKindGitHub {
				t.Fatalf("Kind = %v, want sourceKindGitHub", got.Kind)
			}
			if got.GitHub.Owner != "owner" || got.GitHub.Repo != "repo" {
				t.Errorf("GitHub = %+v, want Owner=%q Repo=%q", got.GitHub, "owner", "repo")
			}
			if got.Ref != "" || got.Path != "" {
				t.Errorf("Ref/Path = %q/%q, want both empty for a plain full-URL source", got.Ref, got.Path)
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
		{"shorthand-shaped but missing a tree-path path component", "owner/repo/tree/main"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSource(tt.src)
			if err != nil {
				t.Fatalf("parseSource() error = %v", err)
			}
			if got.Kind != sourceKindLocal {
				t.Errorf("Kind = %v, want sourceKindLocal for %q", got.Kind, tt.src)
			}
		})
	}
}

func TestParseSource_GitHubTreePath_Shorthand(t *testing.T) {
	got, err := parseSource("owner/repo/tree/main/skills/tdd")
	if err != nil {
		t.Fatalf("parseSource() error = %v", err)
	}
	if got.Kind != sourceKindGitHub {
		t.Fatalf("Kind = %v, want sourceKindGitHub", got.Kind)
	}
	if got.GitHub.Owner != "owner" || got.GitHub.Repo != "repo" {
		t.Errorf("GitHub = %+v, want Owner=%q Repo=%q", got.GitHub, "owner", "repo")
	}
	if got.Ref != "main" {
		t.Errorf("Ref = %q, want %q", got.Ref, "main")
	}
	if got.Path != "skills/tdd" {
		t.Errorf("Path = %q, want %q", got.Path, "skills/tdd")
	}
}

func TestParseSource_GitHubTreePath_FullURL(t *testing.T) {
	tests := []struct {
		name string
		src  string
		ref  string
		path string
	}{
		{"branch ref, nested path", "https://github.com/owner/repo/tree/main/skills/tdd", "main", "skills/tdd"},
		{"trailing slash", "https://github.com/owner/repo/tree/main/skills/tdd/", "main", "skills/tdd"},
		{"tag ref", "https://github.com/owner/repo/tree/v2.1.0/skills/tdd", "v2.1.0", "skills/tdd"},
		{"commit SHA ref", "https://github.com/owner/repo/tree/abcdef0123456789/skills/tdd", "abcdef0123456789", "skills/tdd"},
		{"single-segment path", "https://github.com/owner/repo/tree/main/tdd", "main", "tdd"},
		{"http scheme", "http://github.com/owner/repo/tree/main/skills/tdd", "main", "skills/tdd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSource(tt.src)
			if err != nil {
				t.Fatalf("parseSource() error = %v", err)
			}
			if got.Kind != sourceKindGitHub {
				t.Fatalf("Kind = %v, want sourceKindGitHub", got.Kind)
			}
			if got.GitHub.Owner != "owner" || got.GitHub.Repo != "repo" {
				t.Errorf("GitHub = %+v, want Owner=%q Repo=%q", got.GitHub, "owner", "repo")
			}
			if got.Ref != tt.ref {
				t.Errorf("Ref = %q, want %q", got.Ref, tt.ref)
			}
			if got.Path != tt.path {
				t.Errorf("Path = %q, want %q", got.Path, tt.path)
			}
		})
	}
}

func TestParseSource_GitHubTreePath_InvalidPathSegments_Errors(t *testing.T) {
	tests := []string{
		"owner/repo/tree/main/../escape",
		"owner/repo/tree/main/skills/../../escape",
		"owner/repo/tree/main/skills//tdd",
		"owner/repo/tree/main/./tdd",
	}
	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			_, err := parseSource(src)
			if err == nil {
				t.Fatalf("parseSource(%q) error = nil, want error for an invalid tree-path path", src)
			}
		})
	}
}

func TestParseSource_GitHubTreePath_CombinedWithRefPin_Errors(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"shorthand tree-path with pin", "owner/repo/tree/main/skills/tdd@v2.1.0"},
		{"full-URL tree-path with pin", "https://github.com/owner/repo/tree/main/skills/tdd@v2.1.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSource(tt.src)
			if err == nil {
				t.Fatalf("parseSource(%q) error = nil, want a clear error rejecting the combination", tt.src)
			}
			if !strings.Contains(err.Error(), "@v2.1.0") {
				t.Errorf("error %q does not mention the offending ref pin", err.Error())
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
