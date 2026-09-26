package intelligence

import (
	"crypto/sha256"
	"encoding/hex"
	jsonv2 "encoding/json/v2"
	"net/url"
	"strings"
)

// ProductSignal is explicit inventory evidence supplied by an operator. A
// response banner is always low confidence, regardless of a claimed level.
type ProductSignal struct {
	CPE23, Vendor, Product, Version string
	Part                            string // a, o, h; required for NVD matching without a full CPE
	Method                          string // inventory, manual, banner
	Confidence                      string // high, medium, low
	EvidenceRef                     string // opaque local reference, never a secret or page body
}

// Backport is an explicit operator attestation linked to a vendor advisory.
// The adapter never fetches that URL or infers a backport from a version string.
type Backport struct {
	CVEID, Version, AdvisoryURL, EvidenceRef string
	Confirmed                                bool
}

// Verification is a separate manual or safe-check assertion. There is no
// CVE-specific active check in M2, so none is produced automatically.
type Verification struct {
	Method, EvidenceRef string // manual, safe_check
	Confirmed           bool
}

type Assessment struct {
	CVEID, Source, SourceSHA256     string
	Status                          string // candidate, applicable, verified, not_applicable, unknown
	Confidence                      string // high, medium, low
	Reason                          string // stable, language-independent code
	SignalMethod, SignalEvidenceRef string
	AttestationRef                  string
}

func validRef(v string) bool {
	if len(v) < 1 || len(v) > 128 {
		return false
	}
	for _, r := range v {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == ':' || r == '.') {
			return false
		}
	}
	return true
}

func validSignal(s ProductSignal) bool {
	if s.Version == "" || s.Version == "*" || s.Version == "-" || len(s.Version) > 128 || !validRef(s.EvidenceRef) ||
		(s.Method != "inventory" && s.Method != "manual" && s.Method != "banner") ||
		(s.Confidence != "high" && s.Confidence != "medium" && s.Confidence != "low") {
		return false
	}
	if s.CPE23 == "" {
		return s.Vendor != "" && s.Product != "" && len(s.Vendor) <= 128 && len(s.Product) <= 128 &&
			(s.Part == "" || s.Part == "a" || s.Part == "o" || s.Part == "h")
	}
	cpe, ok := parseCPE(s.CPE23)
	return ok && cpe.version == s.Version && cpe.vendor != "*" && cpe.product != "*" && (s.Part == "" || s.Part == cpe.part)
}

// Assess evaluates one source record against one product signal. Unknown
// version schemes and complex configurations fail to unknown, not unaffected.
func Assess(r Record, s ProductSignal, b *Backport, v *Verification) (Assessment, error) {
	if !validRecord(r) || !validSignal(s) {
		return Assessment{}, ErrInvalid
	}
	h := sha256.Sum256(r.JSON)
	if r.SHA256 != "" && r.SHA256 != hex.EncodeToString(h[:]) {
		return Assessment{}, ErrCorrupt
	}
	if r.Source == "nvd" {
		var meta struct {
			ID string `json:"id"`
		}
		if jsonv2.Unmarshal(r.JSON, &meta) != nil || meta.ID != r.ID {
			return Assessment{}, ErrSource
		}
	} else {
		var meta struct {
			Metadata struct {
				ID string `json:"cveId"`
			} `json:"cveMetadata"`
		}
		if jsonv2.Unmarshal(r.JSON, &meta) != nil || meta.Metadata.ID != r.ID {
			return Assessment{}, ErrSource
		}
	}
	a := Assessment{CVEID: r.ID, Source: r.Source, SourceSHA256: hex.EncodeToString(h[:]), Status: "unknown", Confidence: s.Confidence, Reason: "no_applicability", SignalMethod: s.Method, SignalEvidenceRef: s.EvidenceRef}
	if s.Method == "banner" {
		a.Confidence = "low"
	}
	var result matchResult
	if r.Source == "nvd" {
		if strings.EqualFold(r.State, "Rejected") {
			a.Reason = "source_rejected"
			return a, nil
		}
		result = matchNVD(r.JSON, s)
	} else {
		if r.State != "PUBLISHED" {
			a.Reason = "source_rejected"
			return a, nil
		}
		result = matchCVE(r.JSON, s)
	}
	a.Reason = result.reason
	switch result.value {
	case matchAffected:
		a.Status = "candidate"
		if a.Confidence == "high" && s.Method != "banner" {
			a.Status = "applicable"
		}
	case matchUnaffected:
		if a.Confidence != "high" || s.Method == "banner" {
			a.Reason = "low_confidence_negative"
			return a, nil
		}
		a.Status = "not_applicable"
	default:
		return a, nil
	}
	if b != nil && (a.Status == "applicable" || a.Status == "candidate") && b.CVEID == r.ID && b.Version == s.Version {
		if a.Status != "applicable" {
			a.Status = "unknown"
			a.Reason = "backport_requires_review"
			return a, nil
		}
		if !b.Confirmed || !validRef(b.EvidenceRef) || !validAdvisoryURL(b.AdvisoryURL) {
			a.Status = "unknown"
			a.Reason = "backport_requires_review"
			return a, nil
		}
		a.Status = "not_applicable"
		a.Reason = "backport_attested"
		a.AttestationRef = b.EvidenceRef
		return a, nil
	}
	if v != nil && a.Status == "applicable" && v.Confirmed && validRef(v.EvidenceRef) && (v.Method == "manual" || v.Method == "safe_check") {
		a.Status = "verified"
		a.Reason = "verification_attested"
		a.AttestationRef = v.EvidenceRef
	}
	return a, nil
}

func validAdvisoryURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Fragment == "" && len(raw) <= 2048
}

type matchValue uint8

const (
	matchUnknown matchValue = iota
	matchAffected
	matchUnaffected
)

type matchResult struct {
	value  matchValue
	reason string
}

func matchNVD(data []byte, s ProductSignal) matchResult {
	if s.CPE23 == "" && s.Part == "" {
		return matchResult{matchUnknown, "part_not_identified"}
	}
	var doc struct {
		Configurations []struct {
			Operator string `json:"operator"`
			Negate   bool   `json:"negate"`
			Nodes    []struct {
				Operator string `json:"operator"`
				Negate   bool   `json:"negate"`
				Children []any  `json:"children"`
				CPEMatch []struct {
					Vulnerable     bool   `json:"vulnerable"`
					Criteria       string `json:"criteria"`
					StartIncluding string `json:"versionStartIncluding"`
					StartExcluding string `json:"versionStartExcluding"`
					EndIncluding   string `json:"versionEndIncluding"`
					EndExcluding   string `json:"versionEndExcluding"`
				} `json:"cpeMatch"`
			} `json:"nodes"`
		} `json:"configurations"`
	}
	if jsonv2.Unmarshal(data, &doc) != nil || len(doc.Configurations) == 0 {
		return matchResult{matchUnknown, "no_applicability"}
	}
	matchedProduct := false
	unsupported := false
	negative := false
	positive := false
	for _, cfg := range doc.Configurations {
		if cfg.Negate || cfg.Operator != "" && cfg.Operator != "OR" {
			unsupported = true
			continue
		}
		for _, node := range cfg.Nodes {
			if node.Negate || len(node.Children) != 0 || node.Operator != "OR" {
				unsupported = true
				continue
			}
			for _, criterion := range node.CPEMatch {
				cpe, ok := parseCPE(criterion.Criteria)
				if !ok {
					unsupported = true
					continue
				}
				if !cpeProductMatches(cpe, s) {
					continue
				}
				matchedProduct = true
				if !criterion.Vulnerable {
					unsupported = true
					continue
				}
				decision := matchCPEVersion(cpe.version, s.Version, criterion.StartIncluding, criterion.StartExcluding, criterion.EndIncluding, criterion.EndExcluding)
				switch decision {
				case matchAffected:
					positive = true
				case matchUnaffected:
					negative = true
				default:
					unsupported = true
				}
			}
		}
	}
	if unsupported {
		return matchResult{matchUnknown, "complex_or_unsupported_configuration"}
	}
	if positive {
		return matchResult{matchAffected, "version_in_affected_range"}
	}
	if matchedProduct && negative {
		return matchResult{matchUnaffected, "version_out_of_range"}
	}
	return matchResult{matchUnknown, "product_not_identified"}
}

type cpeFields struct {
	part, vendor, product, version string
	others                         []string
}

func parseCPE(raw string) (cpeFields, bool) {
	if strings.Contains(raw, "\\") {
		return cpeFields{}, false
	} // escaped CPE components need the full CPE matching algorithm
	p := strings.Split(raw, ":")
	if len(p) != 13 || p[0] != "cpe" || p[1] != "2.3" || (p[2] != "a" && p[2] != "o" && p[2] != "h") || p[3] == "" || p[4] == "" {
		return cpeFields{}, false
	}
	return cpeFields{part: p[2], vendor: p[3], product: p[4], version: p[5], others: p[6:]}, true
}

func cpeProductMatches(criteria cpeFields, s ProductSignal) bool {
	if s.CPE23 != "" {
		product, _ := parseCPE(s.CPE23)
		if criteria.part != product.part || criteria.vendor != product.vendor || criteria.product != product.product {
			return false
		}
		for i, c := range criteria.others {
			if c != "*" && c != product.others[i] {
				return false
			}
		}
		return true
	}
	for _, c := range criteria.others {
		if c != "*" {
			return false
		}
	}
	return criteria.part == s.Part && strings.EqualFold(criteria.vendor, s.Vendor) && strings.EqualFold(criteria.product, s.Product)
}

func matchCPEVersion(criteria, actual, startInc, startExc, endInc, endExc string) matchValue {
	if startInc != "" && startExc != "" || endInc != "" && endExc != "" {
		return matchUnknown
	}
	if criteria != "*" && criteria != actual {
		return matchUnaffected
	}
	for _, bound := range []struct {
		v                string
		inclusive, lower bool
	}{{startInc, true, true}, {startExc, false, true}, {endInc, true, false}, {endExc, false, false}} {
		if bound.v == "" {
			continue
		}
		cmp, ok := compareNumeric(actual, bound.v, false)
		if !ok {
			return matchUnknown
		}
		if bound.lower && (cmp < 0 || cmp == 0 && !bound.inclusive) || !bound.lower && (cmp > 0 || cmp == 0 && !bound.inclusive) {
			return matchUnaffected
		}
	}
	return matchAffected
}
