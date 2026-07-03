package nodeguard

import (
	"fmt"
	"strings"
)

type Actual struct {
	Host string
	SNI  string
	Path string
}
type Decision struct {
	Match   bool
	Reasons []string
	Action  string
}

func Evaluate(c Config, a Actual) Decision {
	c = c.Normalize()
	reasons := []string{}
	actualPath := stripMarkerQuery(normalizePath(a.Path))
	if c.ExpectedHost != "" && a.Host != c.ExpectedHost {
		reasons = append(reasons, fmt.Sprintf("host expected=%s actual=%s", c.ExpectedHost, a.Host))
	}
	if c.ExpectedSNI != "" && a.SNI != "" && a.SNI != c.ExpectedSNI {
		reasons = append(reasons, fmt.Sprintf("sni expected=%s actual=%s", c.ExpectedSNI, a.SNI))
	}
	if c.ExpectedPath != "" && actualPath != normalizePath(c.ExpectedPath) && !isLegacyPathAllowed(c.LegacyPaths, actualPath) {
		reasons = append(reasons, fmt.Sprintf("path expected=%s legacy_allowed=%v actual=%s", normalizePath(c.ExpectedPath), c.LegacyPaths, actualPath))
	}
	action := "allow"
	if len(reasons) > 0 {
		if c.Mode == ModeReject {
			action = "reject"
		} else if c.Mode == ModeMonitor {
			action = "monitor"
		}
	}
	return Decision{Match: len(reasons) == 0, Reasons: reasons, Action: action}
}

func isLegacyPathAllowed(paths []string, actual string) bool {
	for _, path := range paths {
		if normalizePath(path) == actual {
			return true
		}
	}
	return false
}

func stripMarkerQuery(path string) string {
	if i := strings.Index(path, "?ng="); i >= 0 {
		return path[:i]
	}
	if i := strings.Index(path, "&ng="); i >= 0 {
		return path[:i]
	}
	if i := strings.Index(path, "?"); i >= 0 {
		return path[:i]
	}
	return path
}
