package reporting

import (
	"bytes"
	jsonv2 "encoding/json/v2"
	"html/template"
	"strings"
	"time"

	"github.com/Matte2599/WebFence/internal/i18n"
	"github.com/Matte2599/WebFence/internal/intelligence"
)

func utc(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func buildReport(s Snapshot, id, language string, now time.Time, signed bool, signerName string, offline bool, ttl time.Duration) (Report, error) {
	if id == "" || (language != "it" && language != "en") || s.ProjectID == "" || s.Run.ID == "" || s.Run.ProjectID != s.ProjectID || s.Run.AuthorizationRevision != s.AuthorizationRevision ||
		s.Run.State == "running" || len(s.Run.Visits) != s.Run.CompletedVisits || now.IsZero() {
		return Report{}, ErrInvalid
	}
	status := "unsigned"
	if signed {
		status = "signed"
	}
	r := Report{Schema: "webfence-report-v1", ReportID: id, Language: language, GeneratedAt: utc(now), SignatureStatus: status, SignerName: signerName,
		Project: ReportProject{ID: s.ProjectID, Name: s.ProjectName, TargetOwner: s.TargetOwner, AuthorizedOrigins: append([]string(nil), s.Origins...),
			AuthorizationRevision: s.AuthorizationRevision, AuthorizationExpiresAt: utc(s.AuthorizationExpiresAt)},
		Run: ReportRun{ID: s.Run.ID, Mode: s.Run.Mode, State: s.Run.State, StartedAtUTC: utc(s.Run.StartedAt), FinishedAtUTC: utc(s.Run.FinishedAt),
			PlannedSeeds: s.Run.PlannedSeeds, CompletedVisits: s.Run.CompletedVisits, RequestsUsed: s.Run.RequestsUsed, StopCode: s.Run.StopCode},
		Coverage: ReportCoverage{Limited: s.Run.CoverageLimited, WholeSiteAttested: false, UnknownUnexecuted: true,
			UnexecutedReason: "whole_site_not_measured"},
		Intelligence: ReportIntelligence{Status: "unavailable"}, Assessments: append([]intelligence.Assessment{}, s.Assessments...), Visits: make([]ReportVisit, 0, len(s.Run.Visits))}
	if s.Run.CoverageLimited || s.Run.State != "complete" {
		r.Coverage.Message = i18n.Text(language, "scan_incomplete")
	} else {
		r.Coverage.Message = i18n.Text(language, "scan_queue_complete")
	}
	if s.Run.PlannedSeeds > s.Run.CompletedVisits {
		r.Coverage.UnexecutedSeedMinimum = s.Run.PlannedSeeds - s.Run.CompletedVisits
	}
	if s.Intelligence != nil {
		r.Intelligence = ReportIntelligence{Status: s.Intelligence.Status(now, ttl, offline), Source: s.Intelligence.Source,
			WindowStart: utc(s.Intelligence.WindowStart), Watermark: utc(s.Intelligence.Watermark),
			LastSuccess: utc(s.Intelligence.LastSuccess), Records: s.Intelligence.Records}
	}
	for _, v := range s.Run.Visits {
		rv := ReportVisit{Index: v.VisitIndex, Depth: v.Depth, HTTPStatus: v.StatusCode, RuleID: v.RuleID, RuleRevision: v.RuleRevision,
			Outcome: v.Outcome, EvidenceCode: v.EvidenceCode, DiscoveryStatus: v.DiscoveryStatus, DiscoveryReason: v.DiscoveryReason,
			Links: v.Links, Forms: v.Forms, OutOfScope: v.OutOfScope, Invalid: v.Invalid,
			Title: i18n.Text(language, "report_rule_"+v.RuleID), OutcomeText: i18n.Text(language, "scan_outcome_"+v.Outcome), EvidenceText: i18n.Text(language, "scan_evidence_"+v.EvidenceCode)}
		switch v.Outcome {
		case "observed":
			r.Coverage.Observed++
		case "not_observed":
			r.Coverage.NotObserved++
		case "inconclusive":
			r.Coverage.Inconclusive++
		case "skipped":
			r.Coverage.Skipped++
		}
		r.Visits = append(r.Visits, rv)
	}
	return r, nil
}

func renderJSON(r Report) ([]byte, error) { return jsonv2.Marshal(r) }

var reportHTML = template.Must(template.New("report").Funcs(template.FuncMap{"tr": func(string) string { return "" }}).Parse(`<!doctype html>
<html lang="{{.Language}}"><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; base-uri 'none'; form-action 'none'"><title>{{tr "report_title"}}</title></head>
<body><main><h1>{{tr "report_title"}}</h1>
<p>{{tr "report_signature"}}: {{tr .SignatureStatus}} · {{tr "report_generated"}}: <time>{{.GeneratedAt}}</time></p>
<h2>{{tr "report_project"}}</h2><p>{{.Project.Name}} ({{.Project.ID}}) · {{.Project.TargetOwner}}</p>
<p>{{tr "report_authorization_revision"}}: {{.Project.AuthorizationRevision}} · {{tr "report_authorization_expires"}}: {{.Project.AuthorizationExpiresAt}}</p>
<ul>{{range .Project.AuthorizedOrigins}}<li>{{.}}</li>{{end}}</ul>
<h2>{{tr "report_run"}}</h2><p>{{.Run.ID}} · {{.Run.State}} · {{.Run.StartedAtUTC}} – {{.Run.FinishedAtUTC}}</p>
<p>{{tr "report_requests"}}: {{.Run.RequestsUsed}} · {{tr "report_visits"}}: {{.Run.CompletedVisits}} · {{tr "report_stop"}}: {{.Run.StopCode}}</p>
<h2>{{tr "report_coverage"}}</h2><p>{{.Coverage.Message}}</p>
<p>{{tr "report_unexecuted"}}: {{.Coverage.UnexecutedSeedMinimum}} {{tr "report_seed_minimum"}}; {{tr "report_remaining_unknown"}}</p>
<p>{{tr "report_observed"}}: {{.Coverage.Observed}} · {{tr "report_not_observed"}}: {{.Coverage.NotObserved}} · {{tr "report_inconclusive"}}: {{.Coverage.Inconclusive}} · {{tr "report_skipped"}}: {{.Coverage.Skipped}}</p>
<h2>{{tr "report_intelligence"}}</h2><p>{{tr .Intelligence.Status}} · {{.Intelligence.Source}} · {{.Intelligence.LastSuccess}}</p>
<h2>{{tr "report_assessments"}}</h2><ul>{{range .Assessments}}<li>{{.CVEID}} · {{.Source}} · {{tr .Status}} · {{.Reason}}</li>{{else}}<li>{{tr "report_none"}}</li>{{end}}</ul>
<h2>{{tr "report_observations"}}</h2><table><thead><tr><th>{{tr "report_visit"}}</th><th>HTTP</th><th>{{tr "report_rule"}}</th><th>{{tr "report_result"}}</th><th>{{tr "report_evidence"}}</th></tr></thead><tbody>
{{range .Visits}}<tr><td>{{.Index}}</td><td>{{.HTTPStatus}}</td><td>{{.Title}} ({{.RuleID}} v{{.RuleRevision}})</td><td>{{.OutcomeText}}</td><td>{{.EvidenceText}} [{{.EvidenceCode}}]</td></tr>{{end}}
</tbody></table><p>{{tr "report_redaction"}}</p></main></body></html>`))

func renderHTML(r Report) ([]byte, error) {
	tmpl, err := reportHTML.Clone()
	if err != nil {
		return nil, err
	}
	tmpl = tmpl.Funcs(template.FuncMap{"tr": func(key string) string {
		if !strings.HasPrefix(key, "report_") {
			key = "report_" + key
		}
		return i18n.Text(r.Language, key)
	}})
	// Static template data is escaped by html/template. Captured pages are never
	// included; only project metadata and the M1 redacted ledger are rendered.
	var out bytes.Buffer
	if err := tmpl.Execute(&out, r); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
