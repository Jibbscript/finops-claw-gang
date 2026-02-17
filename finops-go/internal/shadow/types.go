// Package shadow provides offline comparison of Go and Python FinOps pipeline outputs.
package shadow

// ComparisonResult is the top-level output of a shadow-run comparison.
type ComparisonResult struct {
	Phases    []PhaseComparison `json:"phases"`
	AllMatch  bool              `json:"all_match"`
	Summary   string            `json:"summary"`
}

// PhaseComparison records the comparison for a single pipeline phase.
type PhaseComparison struct {
	Phase     string `json:"phase"`
	GoOutput  string `json:"go_output"`
	PyOutput  string `json:"py_output"`
	Match     bool   `json:"match"`
	DiffLines string `json:"diff_lines,omitempty"`
}
