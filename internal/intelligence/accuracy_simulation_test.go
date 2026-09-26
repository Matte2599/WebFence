package intelligence

import "testing"

// This closed, synthetic corpus measures decisions under stated assumptions.
// Its labels are reviewed inputs, not a sample of real CVE prevalence.
func TestSyntheticAccuracySimulation(t *testing.T) {
	nvd := func(criteria string, extra map[string]any) Record {
		return testRecord(t, "nvd", "Analyzed", nvdMatch(criteria, extra))
	}
	cve := func(affected map[string]any) Record {
		return testRecord(t, "cve", "PUBLISHED", map[string]any{"containers": map[string]any{"cna": map[string]any{"affected": []any{affected}}}})
	}
	product := func(versions []any, defaultStatus string) map[string]any {
		return map[string]any{"vendor": "example", "product": "widget", "versions": versions, "defaultStatus": defaultStatus}
	}
	exact := func(version, status string) []any {
		return []any{map[string]any{"version": version, "status": status}}
	}
	ranged := func() map[string]any {
		return map[string]any{"version": "1.0.0", "lessThan": "2.0.0", "versionType": "semver", "status": "affected"}
	}
	nvdRange := nvd("cpe:2.3:a:example:widget:*:*:*:*:*:*:*:*", map[string]any{"versionStartIncluding": "1.0", "versionEndExcluding": "2.0"})
	nvdExact := nvd("cpe:2.3:a:example:widget:1.5:*:*:*:*:*:*:*", nil)
	cveRange := cve(product([]any{ranged()}, "unaffected"))
	changed := ranged()
	changed["changes"] = []any{map[string]any{"at": "1.5.0", "status": "unaffected"}}
	badChange := ranged()
	badChange["changes"] = []any{map[string]any{"at": "0.5.0", "status": "unaffected"}}
	module := product(exact("1.5.0", "affected"), "unknown")
	module["modules"] = []string{"optional-extension"}
	platform := product(exact("1.5.0", "affected"), "unknown")
	platform["platforms"] = []string{"Windows"}
	packageEntry := product(exact("1.5.0", "affected"), "unknown")
	packageEntry["packageName"] = "widget-package"
	packageEntry["collectionURL"] = "https://packages.example.invalid"
	cpeEntry := product(exact("1.5.0", "affected"), "unknown")
	cpeEntry["cpes"] = []string{"cpe:2.3:a:example:widget:1.5.0:*:*:*:*:*:*:*"}
	conflict := product([]any{
		map[string]any{"version": "1.5.0", "status": "unaffected"}, ranged(),
	}, "unknown")
	same := product([]any{
		map[string]any{"version": "1.5.0", "status": "affected"}, ranged(),
	}, "unknown")
	unsupported := product([]any{map[string]any{"version": "1.0.0", "lessThan": "2.0.0", "versionType": "rpm", "status": "affected"}}, "unknown")
	complexNVD := testRecord(t, "nvd", "Analyzed", map[string]any{"configurations": []any{map[string]any{"nodes": []any{map[string]any{"operator": "AND", "cpeMatch": []any{map[string]any{"vulnerable": true, "criteria": "cpe:2.3:a:example:widget:*:*:*:*:*:*:*:*"}}}}}}})
	type testCase struct {
		name, version, method, confidence, truth, want string
		record                                         Record
	}
	cases := []testCase{
		{"nvd lower inclusive", "1.0", "inventory", "high", "affected", "applicable", nvdRange},
		{"nvd middle", "1.5", "inventory", "high", "affected", "applicable", nvdRange},
		{"nvd below lower", "0.9", "inventory", "high", "unaffected", "not_applicable", nvdRange},
		{"nvd upper exclusive", "2.0", "inventory", "high", "unaffected", "not_applicable", nvdRange},
		{"nvd above upper", "2.1", "inventory", "high", "unaffected", "not_applicable", nvdRange},
		{"nvd exact", "1.5", "inventory", "high", "affected", "applicable", nvdExact},
		{"nvd exact mismatch", "1.6", "inventory", "high", "unaffected", "not_applicable", nvdExact},
		{"nvd weak positive", "1.5", "banner", "high", "affected", "candidate", nvdRange},
		{"nvd weak negative", "2.0", "banner", "high", "unaffected", "unknown", nvdRange},
		{"nvd medium positive", "1.5", "manual", "medium", "affected", "candidate", nvdRange},
		{"nvd unorderable", "1.5-beta", "inventory", "high", "indeterminate", "unknown", nvdRange},
		{"nvd AND", "1.5", "inventory", "high", "indeterminate", "unknown", complexNVD},
		{"cve exact affected", "1.5.0", "inventory", "high", "affected", "applicable", cve(product(exact("1.5.0", "affected"), "unknown"))},
		{"cve exact unaffected", "1.5.0", "inventory", "high", "unaffected", "not_applicable", cve(product(exact("1.5.0", "unaffected"), "unknown"))},
		{"cve range lower", "1.0.0", "inventory", "high", "affected", "applicable", cveRange},
		{"cve range middle", "1.5.0", "inventory", "high", "affected", "applicable", cveRange},
		{"cve range upper", "2.0.0", "inventory", "high", "unaffected", "not_applicable", cveRange},
		{"cve range before", "0.9.0", "inventory", "high", "unaffected", "not_applicable", cveRange},
		{"cve change before", "1.4.0", "inventory", "high", "affected", "applicable", cve(product([]any{changed}, "unknown"))},
		{"cve change boundary", "1.5.0", "inventory", "high", "unaffected", "not_applicable", cve(product([]any{changed}, "unknown"))},
		{"cve change after", "1.6.0", "inventory", "high", "unaffected", "not_applicable", cve(product([]any{changed}, "unknown"))},
		{"cve change outside range", "1.5.0", "inventory", "high", "indeterminate", "unknown", cve(product([]any{badChange}, "unknown"))},
		{"cve prerelease", "1.5.0-rc.1", "inventory", "high", "indeterminate", "unknown", cveRange},
		{"cve rpm scheme", "1.5.0", "inventory", "high", "indeterminate", "unknown", cve(unsupported)},
		{"cve platform constraint", "1.5.0", "inventory", "high", "indeterminate", "unknown", cve(platform)},
		{"cve module constraint", "1.5.0", "inventory", "high", "indeterminate", "unknown", cve(module)},
		{"cve package identity", "1.5.0", "inventory", "high", "indeterminate", "unknown", cve(packageEntry)},
		{"cve CPE qualifier", "1.5.0", "inventory", "high", "indeterminate", "unknown", cve(cpeEntry)},
		{"cve conflicting overlap", "1.5.0", "inventory", "high", "indeterminate", "unknown", cve(conflict)},
		{"cve agreeing overlap", "1.5.0", "inventory", "high", "affected", "applicable", cve(same)},
	}
	var tp, tn, fp, fn, abstain, indeterminate int
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := Assess(tc.record, signal(tc.version, tc.method, tc.confidence), nil, nil)
			if err != nil || a.Status != tc.want {
				t.Fatalf("want %s, got %s (%s), err %v", tc.want, a.Status, a.Reason, err)
			}
			switch tc.truth {
			case "affected":
				switch a.Status {
				case "applicable", "verified":
					tp++
				case "not_applicable":
					fn++
				default:
					abstain++
				}
			case "unaffected":
				switch a.Status {
				case "not_applicable":
					tn++
				case "applicable", "verified":
					fp++
				default:
					abstain++
				}
			case "indeterminate":
				indeterminate++
				if a.Status != "unknown" {
					t.Fatalf("unsupported case must abstain, got %s", a.Status)
				}
			default:
				t.Fatal("invalid simulation label")
			}
		})
	}
	if fp != 0 || fn != 0 {
		t.Fatalf("unsafe conclusive decisions: FP=%d FN=%d", fp, fn)
	}
	t.Logf("synthetic corpus: cases=%d, labeled affected/unaffected=%d, conclusive TP=%d TN=%d FP=%d FN=%d, abstained=%d, indeterminate=%d; conclusive coverage=%d/%d", len(cases), tp+tn+fp+fn+abstain, tp, tn, fp, fn, abstain, indeterminate, tp+tn+fp+fn, tp+tn+fp+fn+abstain)
}

func TestNVDProductPartIsExplicit(t *testing.T) {
	r := testRecord(t, "nvd", "Analyzed", nvdMatch("cpe:2.3:a:example:widget:*:*:*:*:*:*:*:*", nil))
	s := ProductSignal{Vendor: "example", Product: "widget", Version: "1.5", Method: "inventory", Confidence: "high", EvidenceRef: "inventory:42"}
	for _, tc := range []struct{ part, want string }{{"", "unknown"}, {"o", "unknown"}, {"a", "applicable"}} {
		s.Part = tc.part
		a, err := Assess(r, s, nil, nil)
		if err != nil || a.Status != tc.want {
			t.Fatalf("part %q: %+v %v", tc.part, a, err)
		}
	}
}
