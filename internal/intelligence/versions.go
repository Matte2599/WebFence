package intelligence

import (
	jsonv2 "encoding/json/v2"
	"sort"
	"strconv"
	"strings"
)

// compareNumeric covers dotted numeric versions only. SemVer prereleases,
// epochs, distro revisions and ecosystem-specific orderings are unknown here.
func compareNumeric(a, b string, semver bool) (int, bool) {
	parse := func(v string) ([]int64, bool) {
		parts := strings.Split(v, ".")
		if len(parts) < 1 || len(parts) > 8 || semver && len(parts) != 3 {
			return nil, false
		}
		out := make([]int64, len(parts))
		for i, p := range parts {
			if p == "" || len(p) > 12 || semver && len(p) > 1 && p[0] == '0' {
				return nil, false
			}
			for _, r := range p {
				if r < '0' || r > '9' {
					return nil, false
				}
			}
			n, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				return nil, false
			}
			out[i] = n
		}
		return out, true
	}
	x, ok := parse(a)
	if !ok {
		return 0, false
	}
	y, ok := parse(b)
	if !ok {
		return 0, false
	}
	for i := 0; i < len(x) || i < len(y); i++ {
		var a, b int64
		if i < len(x) {
			a = x[i]
		}
		if i < len(y) {
			b = y[i]
		}
		if a < b {
			return -1, true
		}
		if a > b {
			return 1, true
		}
	}
	return 0, true
}

type cveVersion struct {
	Version         string `json:"version"`
	Status          string `json:"status"`
	VersionType     string `json:"versionType"`
	LessThan        string `json:"lessThan"`
	LessThanOrEqual string `json:"lessThanOrEqual"`
	Changes         []struct {
		At     string `json:"at"`
		Status string `json:"status"`
	} `json:"changes"`
}

func matchCVE(data []byte, s ProductSignal) matchResult {
	var doc struct {
		Containers struct {
			CNA struct {
				Affected []struct {
					Vendor        string       `json:"vendor"`
					Product       string       `json:"product"`
					PackageName   string       `json:"packageName"`
					CollectionURL string       `json:"collectionURL"`
					DefaultStatus string       `json:"defaultStatus"`
					Platforms     []string     `json:"platforms"`
					Modules       []string     `json:"modules"`
					CPEs          []string     `json:"cpes"`
					Versions      []cveVersion `json:"versions"`
				} `json:"affected"`
			} `json:"cna"`
		} `json:"containers"`
	}
	if jsonv2.Unmarshal(data, &doc) != nil || len(doc.Containers.CNA.Affected) == 0 {
		return matchResult{matchUnknown, "no_applicability"}
	}
	vendor, product := s.Vendor, s.Product
	if s.CPE23 != "" {
		cpe, _ := parseCPE(s.CPE23)
		vendor, product = cpe.vendor, cpe.product
	}
	matched := false
	var decision matchValue
	unsupported := false
	for _, entry := range doc.Containers.CNA.Affected {
		if !strings.EqualFold(entry.Vendor, vendor) || !strings.EqualFold(entry.Product, product) {
			continue
		}
		matched = true
		if len(entry.Platforms) > 0 || len(entry.Modules) > 0 || len(entry.CPEs) > 0 || entry.PackageName != "" || entry.CollectionURL != "" {
			unsupported = true
			continue
		}
		value, ok := matchCVEVersions(entry.Versions, entry.DefaultStatus, s.Version)
		if !ok {
			unsupported = true
			continue
		}
		if value == matchUnknown {
			unsupported = true
			continue
		}
		if decision != matchUnknown && decision != value {
			unsupported = true
			continue
		}
		decision = value
	}
	if unsupported {
		return matchResult{matchUnknown, "unsupported_version_or_platform"}
	}
	if !matched {
		return matchResult{matchUnknown, "product_not_identified"}
	}
	if decision == matchAffected {
		return matchResult{matchAffected, "version_in_affected_range"}
	}
	if decision == matchUnaffected {
		return matchResult{matchUnaffected, "version_unaffected_by_source"}
	}
	return matchResult{matchUnknown, "version_status_unknown"}
}

func matchCVEVersions(entries []cveVersion, defaultStatus, actual string) (matchValue, bool) {
	decision := matchUnknown
	for _, entry := range entries {
		if entry.Version == "" || (entry.LessThan != "" && entry.LessThanOrEqual != "") {
			return matchUnknown, false
		}
		if entry.LessThan == "" && entry.LessThanOrEqual == "" {
			if entry.Version == actual {
				value := statusValue(entry.Status)
				if value == matchUnknown || decision != matchUnknown && decision != value {
					return matchUnknown, false
				}
				decision = value
			}
			continue
		}
		if entry.VersionType != "semver" {
			return matchUnknown, false
		}
		low, ok := compareNumeric(actual, entry.Version, true)
		if !ok {
			return matchUnknown, false
		}
		if low < 0 {
			continue
		}
		upper := entry.LessThan
		if upper == "" {
			upper = entry.LessThanOrEqual
		}
		if upper != "*" {
			cmp, ok := compareNumeric(actual, upper, true)
			if !ok {
				return matchUnknown, false
			}
			if cmp > 0 || cmp == 0 && entry.LessThan != "" {
				continue
			}
		}
		value := statusValue(entry.Status)
		if value == matchUnknown {
			return matchUnknown, false
		}
		changes := append([]struct {
			At     string `json:"at"`
			Status string `json:"status"`
		}(nil), entry.Changes...)
		if len(changes) > 32 {
			return matchUnknown, false
		}
		for _, change := range changes {
			fromStart, ok := compareNumeric(change.At, entry.Version, true)
			if !ok || fromStart < 0 || statusValue(change.Status) == matchUnknown {
				return matchUnknown, false
			}
			if upper != "*" {
				fromEnd, ok := compareNumeric(change.At, upper, true)
				if !ok || fromEnd > 0 || fromEnd == 0 && entry.LessThan != "" {
					return matchUnknown, false
				}
			}
		}
		sort.Slice(changes, func(i, j int) bool { cmp, _ := compareNumeric(changes[i].At, changes[j].At, true); return cmp < 0 })
		for i := 1; i < len(changes); i++ {
			cmp, _ := compareNumeric(changes[i-1].At, changes[i].At, true)
			if cmp == 0 {
				return matchUnknown, false
			}
		}
		for _, change := range changes {
			cmp, _ := compareNumeric(actual, change.At, true)
			if cmp >= 0 {
				value = statusValue(change.Status)
			}
		}
		if value == matchUnknown || decision != matchUnknown && decision != value {
			return matchUnknown, false
		}
		decision = value
	}
	if decision != matchUnknown {
		return decision, true
	}
	return statusValue(defaultStatus), true
}

func statusValue(status string) matchValue {
	switch status {
	case "affected":
		return matchAffected
	case "unaffected":
		return matchUnaffected
	default:
		return matchUnknown
	}
}
