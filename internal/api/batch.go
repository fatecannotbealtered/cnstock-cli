package api

// aggregateBatch runs fetch for each target in order and assembles the shared
// BatchResult shape. It is the single aggregation contract for both class A
// (the per-target lookup is an in-memory map hit on one multi-code response) and
// class B (the per-target lookup is a real upstream call). A failed item never
// rolls back succeeded items.
//
// continueOnError=true (the default for these read-only queries) finishes the
// whole batch best-effort. continueOnError=false stops at the first failure;
// already-collected items stay, and every still-unattempted target is reported
// in summary.skipped so the agent can resume.
func aggregateBatch[T any](targets []string, continueOnError bool, fetch func(target string) (T, error)) *BatchResult[T] {
	result := &BatchResult[T]{Items: make([]BatchItem[T], 0, len(targets))}
	result.Summary.Total = len(targets)

	for i, target := range targets {
		data, err := fetch(target)
		if err != nil {
			result.Items = append(result.Items, BatchItem[T]{
				Target: target,
				OK:     false,
				Error:  classifyBatchError(err),
			})
			result.Summary.Failed++
			if !continueOnError {
				// Stop early: the remaining targets are unattempted, not failed.
				result.Summary.Skipped = len(targets) - (i + 1)
				return result
			}
			continue
		}
		result.Items = append(result.Items, BatchItem[T]{
			Target: target,
			OK:     true,
			Data:   data,
		})
		result.Summary.Succeeded++
	}
	return result
}
