package skl

import (
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
	if got.Path != "" {
		t.Errorf("Path = %q, want empty for a plain shorthand source", got.Path)
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
			if got.Path != "" {
				t.Errorf("Path = %q, want empty for a plain full-URL source", got.Path)
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
		{"shorthand-shaped but missing a tree-path path component", "owner/repo/tree"},
		{"github.com blob URL (points at a file, not a tree/dir; not a recognized syntax)", "https://github.com/owner/repo/blob/main/skills/tdd/SKILL.md"},
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
	got, err := parseSource("owner/repo/tree/skills/tdd")
	if err != nil {
		t.Fatalf("parseSource() error = %v", err)
	}
	if got.Kind != sourceKindGitHub {
		t.Fatalf("Kind = %v, want sourceKindGitHub", got.Kind)
	}
	if got.GitHub.Owner != "owner" || got.GitHub.Repo != "repo" {
		t.Errorf("GitHub = %+v, want Owner=%q Repo=%q", got.GitHub, "owner", "repo")
	}
	if got.Path != "skills/tdd" {
		t.Errorf("Path = %q, want %q", got.Path, "skills/tdd")
	}
}

func TestParseSource_GitHubTreePath_FullURL(t *testing.T) {
	tests := []struct {
		name string
		src  string
		path string
	}{
		{"nested path", "https://github.com/owner/repo/tree/skills/tdd", "skills/tdd"},
		{"trailing slash", "https://github.com/owner/repo/tree/skills/tdd/", "skills/tdd"},
		{"single-segment path", "https://github.com/owner/repo/tree/tdd", "tdd"},
		{"http scheme", "http://github.com/owner/repo/tree/skills/tdd", "skills/tdd"},
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
			if got.Path != tt.path {
				t.Errorf("Path = %q, want %q", got.Path, tt.path)
			}
		})
	}
}

func TestParseSource_GitHubTreePath_InvalidPathSegments_Errors(t *testing.T) {
	tests := []string{
		"owner/repo/tree/../escape",
		"owner/repo/tree/skills/../../escape",
		"owner/repo/tree/skills//tdd",
		"owner/repo/tree/./tdd",
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

// TestParseSource_GitHubTreePath_RealBrowserURLAbsorbsRefIntoPath documents
// a known v1 limitation (see parsedSource's doc comment): a real GitHub
// tree URL copied from a browser always includes a ref/branch segment
// before the path, but v1's tree-path grammar has no ref concept at all,
// so that segment is absorbed as the first component of Path rather than
// being recognized and stripped. Pasting such a URL as-is therefore names
// the wrong path; the ref segment must be removed by hand until #11 adds
// ref-pinning.
func TestParseSource_GitHubTreePath_RealBrowserURLAbsorbsRefIntoPath(t *testing.T) {
	got, err := parseSource("https://github.com/owner/repo/tree/main/skills/tdd")
	if err != nil {
		t.Fatalf("parseSource() error = %v", err)
	}
	if got.Kind != sourceKindGitHub {
		t.Fatalf("Kind = %v, want sourceKindGitHub", got.Kind)
	}
	if got.Path != "main/skills/tdd" {
		t.Errorf("Path = %q, want %q (the \"main\" ref segment absorbed into the path, not stripped)", got.Path, "main/skills/tdd")
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
