package skl_test

import (
	"reflect"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

func TestResolveAdapters_NoAgentFlagGiven(t *testing.T) {
	tests := []struct {
		name      string
		detected  []string
		want      []string
		wantError bool
	}{
		{
			name:     "zero non-universal adapters detected installs to universal only",
			detected: []string{"universal"},
			want:     []string{"universal"},
		},
		{
			name:     "exactly one non-universal adapter detected installs to universal plus that one",
			detected: []string{"universal", "claude-code"},
			want:     []string{"claude-code", "universal"},
		},
		{
			name:     "exactly one non-universal adapter detected (kit) installs to universal plus kit",
			detected: []string{"universal", "kit"},
			want:     []string{"kit", "universal"},
		},
		{
			name:      "two or more non-universal adapters detected errors instead of guessing",
			detected:  []string{"universal", "claude-code", "kit"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := skl.ResolveAdapters(nil, tt.detected)
			if tt.wantError {
				if err == nil {
					t.Fatalf("ResolveAdapters() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveAdapters() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveAdapters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveAdapters_AmbiguousDefaultErrorListsAllDetectedAdapters(t *testing.T) {
	_, err := skl.ResolveAdapters(nil, []string{"universal", "claude-code", "kit"})
	if err == nil {
		t.Fatalf("ResolveAdapters() error = nil, want error")
	}
	for _, want := range []string{"universal", "claude-code", "kit"} {
		if !contains(err.Error(), want) {
			t.Errorf("error %q does not mention detected adapter %q", err.Error(), want)
		}
	}
}

func TestResolveAdapters_ExplicitNamedAdapters(t *testing.T) {
	detected := []string{"universal", "claude-code", "kit"}

	got, err := skl.ResolveAdapters([]string{"claude-code"}, detected)
	if err != nil {
		t.Fatalf("ResolveAdapters() error = %v", err)
	}
	if want := []string{"claude-code"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ResolveAdapters() = %v, want %v (explicit naming targets only what's named)", got, want)
	}
}

func TestResolveAdapters_UnknownAdapterNameErrors(t *testing.T) {
	detected := []string{"universal"}

	_, err := skl.ResolveAdapters([]string{"not-a-real-adapter"}, detected)
	if err == nil {
		t.Fatalf("ResolveAdapters() error = nil, want error for unknown adapter name")
	}
	if !contains(err.Error(), "universal") {
		t.Errorf("error %q does not list detected adapters as suggestions", err.Error())
	}
}

func TestResolveAdapters_KnownButUndetectedAdapterErrors(t *testing.T) {
	// kit is a known adapter name, but not present in the detected set.
	detected := []string{"universal", "claude-code"}

	_, err := skl.ResolveAdapters([]string{"kit"}, detected)
	if err == nil {
		t.Fatalf("ResolveAdapters() error = nil, want error for known-but-undetected adapter")
	}
	for _, want := range []string{"universal", "claude-code"} {
		if !contains(err.Error(), want) {
			t.Errorf("error %q does not list detected adapter %q as a suggestion", err.Error(), want)
		}
	}
}

func TestResolveAdapters_CommaSeparatedAndRepeatedFlagsAreEquivalent(t *testing.T) {
	detected := []string{"universal", "claude-code", "kit"}

	commaSeparated, err := skl.ResolveAdapters([]string{"claude-code,kit"}, detected)
	if err != nil {
		t.Fatalf("ResolveAdapters() comma-separated error = %v", err)
	}

	repeated, err := skl.ResolveAdapters([]string{"claude-code", "kit"}, detected)
	if err != nil {
		t.Fatalf("ResolveAdapters() repeated-flag error = %v", err)
	}

	if !reflect.DeepEqual(commaSeparated, repeated) {
		t.Errorf("comma-separated result %v != repeated-flag result %v", commaSeparated, repeated)
	}
	if want := []string{"claude-code", "kit"}; !reflect.DeepEqual(commaSeparated, want) {
		t.Errorf("ResolveAdapters() = %v, want %v", commaSeparated, want)
	}
}

func TestResolveAdapters_WildcardInstallsToUniversalPlusAllDetected(t *testing.T) {
	detected := []string{"universal", "claude-code", "kit"}

	got, err := skl.ResolveAdapters([]string{"*"}, detected)
	if err != nil {
		t.Fatalf("ResolveAdapters() error = %v, want no error even with 2+ non-universal adapters detected", err)
	}
	want := []string{"claude-code", "kit", "universal"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ResolveAdapters() = %v, want %v", got, want)
	}
}

func TestResolveAdapters_WildcardNeverTargetsUndetectedAdapter(t *testing.T) {
	// kit is not present in the detected set at all.
	detected := []string{"universal", "claude-code"}

	got, err := skl.ResolveAdapters([]string{"*"}, detected)
	if err != nil {
		t.Fatalf("ResolveAdapters() error = %v", err)
	}
	if contains(joinStrings(got), "kit") {
		t.Errorf("ResolveAdapters() = %v, should never include undetected adapter %q", got, "kit")
	}
	want := []string{"claude-code", "universal"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ResolveAdapters() = %v, want %v", got, want)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) != -1
}

func joinStrings(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}
