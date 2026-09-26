package intelligence

import (
	"encoding/json"
	"testing"
	"time"
)

func testRecord(t *testing.T, source, state string, body any) Record {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	if source == "nvd" {
		object["id"] = "CVE-2026-1000"
	} else {
		object["cveMetadata"] = map[string]any{"cveId": "CVE-2026-1000"}
	}
	data, err = json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return Record{Source: source, ID: "CVE-2026-1000", State: state, Modified: time.Now().UTC(), Acquired: time.Now().UTC(), JSON: data}
}

func nvdMatch(criteria string, extra map[string]any) map[string]any {
	m := map[string]any{"vulnerable": true, "criteria": criteria}
	for k, v := range extra {
		m[k] = v
	}
	return map[string]any{"configurations": []any{map[string]any{"nodes": []any{map[string]any{"operator": "OR", "cpeMatch": []any{m}}}}}}
}

func signal(version, method, confidence string) ProductSignal {
	return ProductSignal{CPE23: "cpe:2.3:a:example:widget:" + version + ":*:*:*:*:*:*:*", Version: version, Method: method, Confidence: confidence, EvidenceRef: "inventory:42"}
}

func TestNVDAssessmentConfidenceBackportAndVerification(t *testing.T) {
	record := testRecord(t, "nvd", "Analyzed", nvdMatch("cpe:2.3:a:example:widget:*:*:*:*:*:*:*:*", map[string]any{"versionStartIncluding": "1.0", "versionEndExcluding": "2.0"}))
	assert := func(s ProductSignal, b *Backport, v *Verification, want string) {
		t.Helper()
		a, err := Assess(record, s, b, v)
		if err != nil || a.Status != want || a.SourceSHA256 == "" || a.CVEID != record.ID {
			t.Fatalf("assessment %s: %+v %v", want, a, err)
		}
	}
	assert(signal("1.5", "inventory", "high"), nil, nil, "applicable")
	assert(signal("1.5", "banner", "high"), nil, nil, "candidate")
	assert(signal("1.5", "inventory", "medium"), nil, nil, "candidate")
	assert(signal("2.0", "inventory", "high"), nil, nil, "not_applicable")
	assert(signal("2.0", "banner", "high"), nil, nil, "unknown")
	assert(signal("1.5", "inventory", "high"), nil, &Verification{Method: "manual", EvidenceRef: "review:7", Confirmed: true}, "verified")
	assert(signal("1.5", "banner", "high"), nil, &Verification{Method: "manual", EvidenceRef: "review:7", Confirmed: true}, "candidate")
	backport := &Backport{CVEID: record.ID, Version: "1.5", AdvisoryURL: "https://vendor.example/advisory/42", EvidenceRef: "advisory:42", Confirmed: true}
	assert(signal("1.5", "inventory", "high"), backport, nil, "not_applicable")
	backport.Confirmed = false
	assert(signal("1.5", "inventory", "high"), backport, nil, "unknown")
	backport.Confirmed = true
	assert(signal("1.5", "banner", "high"), backport, nil, "unknown")
}

func TestNVDUnsupportedAndRejectedStayUnknown(t *testing.T) {
	cases := []struct {
		name, state, version, operator string
		extra                          map[string]any
	}{
		{"rejected", "Rejected", "1.5", "OR", nil},
		{"complex", "Analyzed", "1.5", "AND", nil},
		{"unorderable", "Analyzed", "1.5-beta", "OR", map[string]any{"versionEndExcluding": "2.0"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := nvdMatch("cpe:2.3:a:example:widget:*:*:*:*:*:*:*:*", tc.extra)
			if tc.operator == "AND" {
				body["configurations"].([]any)[0].(map[string]any)["nodes"].([]any)[0].(map[string]any)["operator"] = "AND"
			}
			r := testRecord(t, "nvd", tc.state, body)
			a, err := Assess(r, signal(tc.version, "inventory", "high"), nil, nil)
			if err != nil || a.Status != "unknown" {
				t.Fatalf("%+v %v", a, err)
			}
		})
	}
}

func TestCVEVersionChangesAndUnknownSchemes(t *testing.T) {
	base := func(versionType string) Record {
		return testRecord(t, "cve", "PUBLISHED", map[string]any{"containers": map[string]any{"cna": map[string]any{"affected": []any{map[string]any{
			"vendor": "example", "product": "widget", "defaultStatus": "unknown", "versions": []any{map[string]any{
				"version": "1.0.0", "lessThan": "2.0.0", "versionType": versionType, "status": "affected",
				"changes": []any{map[string]any{"at": "1.5.0", "status": "unaffected"}},
			}}}}}}})
	}
	for _, tc := range []struct{ version, want string }{{"1.4.0", "applicable"}, {"1.5.0", "not_applicable"}, {"1.6.0", "not_applicable"}, {"2.0.0", "unknown"}} {
		r := base("semver")
		a, err := Assess(r, signal(tc.version, "inventory", "high"), nil, nil)
		if err != nil || a.Status != tc.want {
			t.Fatalf("%s: %+v %v", tc.version, a, err)
		}
	}
	a, err := Assess(base("rpm"), signal("1.4.0", "inventory", "high"), nil, nil)
	if err != nil || a.Status != "unknown" {
		t.Fatalf("rpm: %+v %v", a, err)
	}
	a, err = Assess(base("semver"), signal("1.4.0-rc.1", "inventory", "high"), nil, nil)
	if err != nil || a.Status != "unknown" {
		t.Fatalf("prerelease: %+v %v", a, err)
	}
}

func TestInvalidSignalAndSourceHash(t *testing.T) {
	r := testRecord(t, "nvd", "Analyzed", nvdMatch("cpe:2.3:a:example:widget:*:*:*:*:*:*:*:*", nil))
	r.SHA256 = "wrong"
	if _, err := Assess(r, signal("1.0", "inventory", "high"), nil, nil); err != ErrCorrupt {
		t.Fatal(err)
	}
	r.SHA256 = ""
	s := signal("1.0", "inventory", "high")
	s.EvidenceRef = "secret/value"
	if _, err := Assess(r, s, nil, nil); err != ErrInvalid {
		t.Fatal(err)
	}
	r.JSON = []byte(`{"id":"CVE-2026-9999","configurations":[]}`)
	if _, err := Assess(r, signal("1.0", "inventory", "high"), nil, nil); err != ErrSource {
		t.Fatalf("wrong source ID: %v", err)
	}
	r.JSON = []byte(`{"id":"CVE-2026-1000","id":"CVE-2026-1000","configurations":[]}`)
	if _, err := Assess(r, signal("1.0", "inventory", "high"), nil, nil); err != ErrSource {
		t.Fatalf("duplicate JSON member: %v", err)
	}
}

func TestComplexConfigurationCannotBeOverriddenBySimpleMatch(t *testing.T) {
	matching := map[string]any{"operator": "OR", "cpeMatch": []any{map[string]any{"vulnerable": true, "criteria": "cpe:2.3:a:example:widget:*:*:*:*:*:*:*:*"}}}
	complex := map[string]any{"operator": "AND", "cpeMatch": []any{map[string]any{"vulnerable": true, "criteria": "cpe:2.3:a:example:widget:*:*:*:*:*:*:*:*"}}}
	r := testRecord(t, "nvd", "Analyzed", map[string]any{"configurations": []any{map[string]any{"nodes": []any{matching, complex}}}})
	a, err := Assess(r, signal("1.0", "inventory", "high"), nil, nil)
	if err != nil || a.Status != "unknown" {
		t.Fatalf("complex result: %+v %v", a, err)
	}
}

func TestConflictingCVEChangesRemainUnknown(t *testing.T) {
	r := testRecord(t, "cve", "PUBLISHED", map[string]any{"containers": map[string]any{"cna": map[string]any{"affected": []any{map[string]any{
		"vendor": "example", "product": "widget", "versions": []any{map[string]any{
			"version": "1.0.0", "lessThan": "2.0.0", "versionType": "semver", "status": "affected",
			"changes": []any{map[string]any{"at": "1.5.0", "status": "affected"}, map[string]any{"at": "1.5.0", "status": "unaffected"}},
		}}}}}}})
	a, err := Assess(r, signal("1.6.0", "inventory", "high"), nil, nil)
	if err != nil || a.Status != "unknown" {
		t.Fatalf("conflicting changes: %+v %v", a, err)
	}
}
