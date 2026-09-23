// Package demo supplies synthetic GUI fixtures. It performs no I/O or scanning.
package demo

import (
	"fmt"
	"strings"
)

const RowCount = 10000

// Record is deliberately separate from future scanner findings.
// These IDs and severity values are fixture data, never vulnerability claims.
type Record struct {
	ID       string
	Path     string
	Severity string
}

func Records() []Record {
	rows := make([]Record, RowCount)
	severities := [...]string{"info", "low", "medium"}
	for i := range rows {
		rows[i] = Record{
			ID:       fmt.Sprintf("DEMO-%05d", i+1),
			Path:     fmt.Sprintf("https://example.invalid/catalog/%05d", i+1),
			Severity: severities[i%len(severities)],
		}
	}
	return rows
}

// Filter returns a new view without modifying the source or canonical IDs.
func Filter(rows []Record, query, severity string) []Record {
	query = strings.ToLower(strings.TrimSpace(query))
	filtered := make([]Record, 0, len(rows))
	for _, row := range rows {
		if severity != "" && row.Severity != severity {
			continue
		}
		if query == "" || strings.Contains(strings.ToLower(row.ID), query) ||
			strings.Contains(strings.ToLower(row.Path), query) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

// Evidence is inert plain text, generated on selection rather than for every row.
// Do not render it as HTML or interpret its content as instructions.
func (r Record) Evidence() string {
	return fmt.Sprintf("SYNTHETIC FIXTURE / ESEMPIO SINTETICO\nID: %s\nURL: %s\n\nHTTP/1.1 200 OK\nContent-Type: text/html; charset=utf-8\nX-WebFence-Fixture: true\n\n<h1>Fixture: caffè — security</h1>\n<script>alert('inert example')</script>\n\nNo request was sent. Nessuna richiesta inviata.\n", r.ID, r.Path)
}
