package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// A POST /jq batch runs under one wall-clock budget. Once it is spent, the
// remaining items are answered with a per-item error without being evaluated,
// so 200 evals × 250 ms can no longer hold a request for 50 seconds.
func TestJQEvalOneRefusesWhenBatchBudgetSpent(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	item := &jqEvalItem{Program: ".", Input: json.RawMessage(`1`)}
	res, ok := jqEvalOne(ctx, item, false).(jqErrorItem)
	if !ok {
		t.Fatalf("expected an error item, got %T", res)
	}
	if !strings.Contains(res.Error, "budget") {
		t.Errorf("expected the error to name the batch budget, got %q", res.Error)
	}
}
