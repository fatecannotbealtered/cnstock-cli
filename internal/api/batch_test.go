package api

import "testing"

// TestAggregateBatch_AllSucceed verifies order preservation and summary tally
// for an all-success batch.
func TestAggregateBatch_AllSucceed(t *testing.T) {
	targets := []string{"a", "b", "c"}
	result := aggregateBatch(targets, true, func(target string) (string, error) {
		return "val-" + target, nil
	})
	if result.Summary.Total != 3 || result.Summary.Succeeded != 3 || result.Summary.Failed != 0 {
		t.Fatalf("summary = %+v, want total=3 succeeded=3 failed=0", result.Summary)
	}
	for i, want := range targets {
		if result.Items[i].Target != want {
			t.Errorf("items[%d].target = %q, want %q (order not preserved)", i, result.Items[i].Target, want)
		}
		if !result.Items[i].OK || result.Items[i].Data != "val-"+want {
			t.Errorf("items[%d] = %+v, want ok data val-%s", i, result.Items[i], want)
		}
	}
}

// TestAggregateBatch_PartialFailureContinues verifies a failed item is captured
// per-item with the right code and does not abort the rest (continue-on-error).
func TestAggregateBatch_PartialFailureContinues(t *testing.T) {
	targets := []string{"ok1", "bad", "ok2"}
	result := aggregateBatch(targets, true, func(target string) (string, error) {
		if target == "bad" {
			return "", newNotFoundError("no data for %s", target)
		}
		return "ok", nil
	})
	if result.Summary.Total != 3 || result.Summary.Succeeded != 2 || result.Summary.Failed != 1 {
		t.Fatalf("summary = %+v, want total=3 succeeded=2 failed=1", result.Summary)
	}
	if result.Summary.Skipped != 0 {
		t.Errorf("skipped = %d, want 0 (continue-on-error finishes the batch)", result.Summary.Skipped)
	}
	bad := result.Items[1]
	if bad.Target != "bad" || bad.OK {
		t.Fatalf("items[1] = %+v, want failed bad", bad)
	}
	if bad.Error == nil || bad.Error.Code != "E_NOT_FOUND" || bad.Error.Retryable {
		t.Errorf("items[1].error = %+v, want E_NOT_FOUND retryable=false", bad.Error)
	}
}

// TestAggregateBatch_StopOnFirstFailure verifies continue-on-error=false stops
// at the first failure, keeps the succeeded item, and reports the rest skipped.
func TestAggregateBatch_StopOnFirstFailure(t *testing.T) {
	targets := []string{"ok1", "bad", "never"}
	attempts := 0
	result := aggregateBatch(targets, false, func(target string) (string, error) {
		attempts++
		if target == "bad" {
			return "", newServerError("boom")
		}
		return "ok", nil
	})
	if attempts != 2 {
		t.Fatalf("attempted %d targets, want 2 (stop after first failure)", attempts)
	}
	if result.Summary.Total != 3 || result.Summary.Succeeded != 1 || result.Summary.Failed != 1 || result.Summary.Skipped != 1 {
		t.Fatalf("summary = %+v, want total=3 succeeded=1 failed=1 skipped=1", result.Summary)
	}
	if len(result.Items) != 2 {
		t.Errorf("items length = %d, want 2 (unattempted target is not an item)", len(result.Items))
	}
	if result.Items[1].Error == nil || result.Items[1].Error.Code != "E_SERVER" || !result.Items[1].Error.Retryable {
		t.Errorf("items[1].error = %+v, want E_SERVER retryable=true", result.Items[1].Error)
	}
}

// TestParseSymbolList_DedupPreservesOrder verifies dedup + first-seen order and
// the empty-input usage error.
func TestParseSymbolList_DedupPreservesOrder(t *testing.T) {
	got, err := ParseSymbolList("600519, 000001 ,600519, hk00700")
	if err != nil {
		t.Fatalf("ParseSymbolList error: %v", err)
	}
	want := []string{"sh600519", "sz000001", "hk00700"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v (dedup failed)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q (order not preserved)", i, got[i], want[i])
		}
	}

	if _, err := ParseSymbolList("  ,  ,"); err == nil {
		t.Error("expected validation error for empty target list, got nil")
	}
}

// TestSingleSymbol_RejectsMultiple verifies the deferred multi-symbol guard for
// minute: a single symbol passes; more than one is a validation error.
func TestSingleSymbol_RejectsMultiple(t *testing.T) {
	sym, err := SingleSymbol("600519")
	if err != nil || sym != "sh600519" {
		t.Fatalf("SingleSymbol(600519) = %q, %v; want sh600519, nil", sym, err)
	}
	if _, err := SingleSymbol("600519,000001"); err == nil {
		t.Error("expected validation error for multi-symbol minute input, got nil")
	} else if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}
