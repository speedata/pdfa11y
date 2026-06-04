package engine

// PromoteWarnings rewrites every Warning finding in results to Error
// severity in place. Used by the CLI's --strict mode to make warnings
// affect the conformance verdict and exit code. Info findings are left
// alone -- they represent advisories, not violations.
func PromoteWarnings(results []Result) {
	for i := range results {
		for j := range results[i].Findings {
			if results[i].Findings[j].Severity == SeverityWarning {
				results[i].Findings[j].Severity = SeverityError
			}
		}
	}
}
