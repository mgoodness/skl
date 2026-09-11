package skl

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveAdapters computes the set of adapter names an Add call should
// target, given the raw --agent flag values (which may be comma-separated,
// repeated, both, or contain the literal "*") and the adapters detected on
// this machine (as returned by DetectedAdapters).
//
// Resolution follows the no-silent-guessing rule that governs --skill too:
// genuine ambiguity is always an error, never a default guess or a prompt.
//
//   - No --agent given: 0 non-universal adapters detected -> {universal};
//     exactly 1 -> {universal, that one}; 2+ -> error, listing the
//     detected adapters as a pick-list.
//   - --agent '*' (alone or combined with anything else): {universal} ∪
//     detected adapters. Never errors, and never targets an undetected
//     adapter.
//   - --agent <name>[,<name>...]: every named adapter must be both known
//     and detected; an unknown or known-but-undetected name is a hard
//     error listing the detected adapters as suggested values.
//
// The returned slice is sorted and de-duplicated.
func ResolveAdapters(requestedAgentValues []string, detected []string) ([]string, error) {
	requested := normalizeAdapterValues(requestedAgentValues)
	detectedSorted := sortedCopy(detected)

	for _, name := range requested {
		if name == "*" {
			return detectedSorted, nil
		}
	}

	if len(requested) == 0 {
		return defaultAdapters(detectedSorted)
	}

	return explicitAdapters(requested, detected, detectedSorted)
}

// defaultAdapters implements the 0/1/2+ heuristic used when --agent is
// omitted entirely. detectedSorted is expected to already be sorted (it
// always includes UniversalAdapter, per DetectedAdapters).
func defaultAdapters(detectedSorted []string) ([]string, error) {
	nonUniversal := make([]string, 0, len(detectedSorted))
	for _, d := range detectedSorted {
		if d != UniversalAdapter {
			nonUniversal = append(nonUniversal, d)
		}
	}

	switch len(nonUniversal) {
	case 0:
		return []string{UniversalAdapter}, nil
	case 1:
		return sortedCopy([]string{UniversalAdapter, nonUniversal[0]}), nil
	default:
		return nil, fmt.Errorf(
			"multiple adapters detected (%s); specify --agent to choose one or more, or --agent '*' for all detected adapters",
			strings.Join(detectedSorted, ", "),
		)
	}
}

// explicitAdapters validates and resolves a non-empty, non-"*" --agent
// value list. Naming an adapter targets exactly that adapter — unlike the
// default heuristic and "*", it does not implicitly add UniversalAdapter.
func explicitAdapters(requested, detected, detectedSorted []string) ([]string, error) {
	knownNames := make(map[string]bool, len(Adapters))
	for _, a := range Adapters {
		knownNames[a.Name] = true
	}
	detectedSet := make(map[string]bool, len(detected))
	for _, d := range detected {
		detectedSet[d] = true
	}

	seen := make(map[string]bool, len(requested))
	result := make([]string, 0, len(requested))
	for _, name := range requested {
		if !knownNames[name] || !detectedSet[name] {
			return nil, fmt.Errorf(
				"unknown or undetected adapter %q; detected adapters: %s",
				name, strings.Join(detectedSorted, ", "),
			)
		}
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result, nil
}

// normalizeAdapterValues flattens a raw --agent value list (which may mix
// comma-separated entries and repeated-flag entries) into individual,
// trimmed adapter names, preserving first-seen order.
func normalizeAdapterValues(raw []string) []string {
	var out []string
	for _, r := range raw {
		for _, part := range strings.Split(r, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			out = append(out, part)
		}
	}
	return out
}

func sortedCopy(s []string) []string {
	out := append([]string{}, s...)
	sort.Strings(out)
	return out
}
