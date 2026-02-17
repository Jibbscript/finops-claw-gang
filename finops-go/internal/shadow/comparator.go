package shadow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Compare compares Go and Python pipeline outputs and produces a ComparisonResult.
func Compare(goJSON, pyJSON []byte) (*ComparisonResult, error) {
	var goState, pyState map[string]any
	if err := json.Unmarshal(goJSON, &goState); err != nil {
		return nil, fmt.Errorf("parse Go output: %w", err)
	}
	if err := json.Unmarshal(pyJSON, &pyState); err != nil {
		return nil, fmt.Errorf("parse Python output: %w", err)
	}

	phases := []string{"triage", "analysis", "approval"}
	var comparisons []PhaseComparison
	allMatch := true

	for _, phase := range phases {
		goVal, _ := json.MarshalIndent(goState[phase], "", "  ") // safe: values came from Unmarshal
		pyVal, _ := json.MarshalIndent(pyState[phase], "", "  ") // safe: values came from Unmarshal

		match := string(goVal) == string(pyVal)
		if !match {
			allMatch = false
		}

		pc := PhaseComparison{
			Phase:    phase,
			GoOutput: string(goVal),
			PyOutput: string(pyVal),
			Match:    match,
		}
		if !match {
			pc.DiffLines = simpleDiff(string(goVal), string(pyVal))
		}
		comparisons = append(comparisons, pc)
	}

	summary := "all phases match"
	if !allMatch {
		var divergent []string
		for _, c := range comparisons {
			if !c.Match {
				divergent = append(divergent, c.Phase)
			}
		}
		summary = fmt.Sprintf("divergence in: %s", strings.Join(divergent, ", "))
	}

	return &ComparisonResult{
		Phases:   comparisons,
		AllMatch: allMatch,
		Summary:  summary,
	}, nil
}

// simpleDiff returns a basic line-by-line diff indicator.
func simpleDiff(a, b string) string {
	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")
	var diffs []string

	maxLen := len(aLines)
	if len(bLines) > maxLen {
		maxLen = len(bLines)
	}

	for i := range maxLen {
		aLine := ""
		if i < len(aLines) {
			aLine = aLines[i]
		}
		bLine := ""
		if i < len(bLines) {
			bLine = bLines[i]
		}
		if aLine != bLine {
			diffs = append(diffs, fmt.Sprintf("line %d:\n  go: %s\n  py: %s", i+1, aLine, bLine))
		}
	}
	return strings.Join(diffs, "\n")
}
