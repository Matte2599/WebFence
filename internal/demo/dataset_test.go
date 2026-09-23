package demo

import (
	"strings"
	"testing"
)

func TestFilterPreservesIdentityAndDataset(t *testing.T) {
	rows := Records()
	if len(rows) != RowCount {
		t.Fatalf("got %d rows", len(rows))
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		if seen[row.ID] {
			t.Fatalf("duplicate ID %q", row.ID)
		}
		seen[row.ID] = true
		if !strings.HasPrefix(row.Path, "https://example.invalid/") {
			t.Fatalf("unexpected fixture URL: %q", row.Path)
		}
	}
	filtered := Filter(rows, "  demo-10000  ", "info")
	if len(filtered) != 1 || filtered[0] != rows[9999] {
		t.Fatalf("wrong final row: %v", filtered)
	}
	if got := Filter(rows, "demo-10000", "medium"); len(got) != 0 {
		t.Fatalf("filter must intersect: %v", got)
	}
	filtered[0].ID = "changed"
	if rows[9999].ID != "DEMO-10000" {
		t.Fatal("filter mutated source")
	}
	if len(Filter(rows, "/catalog/00001", "")) != 1 {
		t.Fatal("URL search failed")
	}
	if len(Filter(nil, "", "")) != 0 {
		t.Fatal("empty input returned rows")
	}
}
