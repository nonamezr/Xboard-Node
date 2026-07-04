package journalguard

import (
	"regexp"
	"strings"
	"time"

	"github.com/cedar2025/xboard-node/internal/nodeguard"
)

var (
	badHostRe      = regexp.MustCompile(`(?i)bad host:\s*([^\s,;]+)`)
	validateHostRe = regexp.MustCompile(`(?i)failed to validate host,\s*request:([^\s,;]+),\s*config:([^\s,;]+)`)
	pathRe         = regexp.MustCompile(`(?i)\bpath[:=]\s*([^\s,;]+)`)
	sniRe          = regexp.MustCompile(`(?i)\bsni[:=]\s*([^\s,;]+)`)
)

type ParsedReject struct {
	ActualHost string
	ActualPath string
	ActualSNI  string
	RawReason  string
	ReportedAt time.Time
}

func ParseRejectLine(line string, now time.Time) (ParsedReject, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return ParsedReject{}, false
	}

	var p ParsedReject
	p.RawReason = line
	p.ReportedAt = parseJournalTimestamp(line, now)

	if m := badHostRe.FindStringSubmatch(line); len(m) == 2 {
		p.ActualHost = cleanHost(m[1])
	}
	if m := validateHostRe.FindStringSubmatch(line); len(m) == 3 {
		p.ActualHost = cleanHost(m[1])
	}
	if m := pathRe.FindStringSubmatch(line); len(m) == 2 {
		p.ActualPath = cleanPath(m[1])
	}
	if m := sniRe.FindStringSubmatch(line); len(m) == 2 {
		p.ActualSNI = cleanHost(m[1])
	}

	if p.ActualHost == "" && p.ActualPath == "" && p.ActualSNI == "" {
		return ParsedReject{}, false
	}
	return p, true
}

func (p ParsedReject) Payload(token string, nodeID int, transport string) nodeguard.ReportPayload {
	return nodeguard.ReportPayload{
		Token:      token,
		NodeID:     nodeID,
		ActualHost: p.ActualHost,
		ActualPath: p.ActualPath,
		ActualSNI:  p.ActualSNI,
		RawReason:  p.RawReason,
		Action:     "monitor",
		Confidence: "low",
		Transport:  transport,
		ReportedAt: p.ReportedAt.Unix(),
	}
}

func cleanHost(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `"'[](),;`)
	if i := strings.IndexByte(v, '/'); i >= 0 {
		v = v[:i]
	}
	if h, _, ok := strings.Cut(v, ":"); ok && h != "" {
		return strings.ToLower(strings.Trim(h, `"'[](),;`))
	}
	return strings.ToLower(v)
}

func cleanPath(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `"'(),;`)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "/") {
		return "/" + v
	}
	return v
}

func parseJournalTimestamp(line string, now time.Time) time.Time {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return now
	}
	candidates := []string{fields[0]}
	if len(fields) >= 2 {
		candidates = append(candidates, fields[0]+" "+fields[1])
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05-07:00", "2006-01-02T15:04:05Z07:00", "2006-01-02 15:04:05"}
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		for _, layout := range layouts {
			if ts, err := time.Parse(layout, c); err == nil {
				return ts
			}
		}
	}
	return now
}
