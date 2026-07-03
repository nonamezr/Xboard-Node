package nodeguard

import "strings"

type Mode string

const (
	ModeOff     Mode = "off"
	ModeMonitor Mode = "monitor_only"
	ModeReject  Mode = "reject_only"
)

type Transport string

const (
	TransportTCPHTTP Transport = "tcp_http"
	TransportWSTLS   Transport = "ws_tls"
)

type Config struct {
	Mode         Mode
	Transport    Transport
	Listen       string
	Backend      string
	ExpectedHost string
	ExpectedSNI  string
	ExpectedPath string
	LegacyPaths  []string
	PanelURL     string
	ServerToken  string
	NodeID       int
	UserID       int
	Email        string
	Protocol     string
	Confidence   string
	TLSCertFile  string
	TLSKeyFile   string
}

func (c Config) Normalize() Config {
	if c.Mode == "" {
		c.Mode = ModeMonitor
	}
	if c.ExpectedPath == "" {
		c.ExpectedPath = "/"
	}
	if len(c.LegacyPaths) == 0 {
		c.LegacyPaths = []string{"/", "/vpn"}
	}
	for i, path := range c.LegacyPaths {
		c.LegacyPaths[i] = normalizePath(path)
	}
	if c.Confidence == "" {
		if c.UserID > 0 || c.Email != "" {
			c.Confidence = "high"
		} else {
			c.Confidence = "medium"
		}
	}
	c.PanelURL = strings.TrimRight(c.PanelURL, "/")
	return c
}

func (m Mode) Valid() bool { return m == ModeOff || m == ModeMonitor || m == ModeReject }

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func SplitLegacyPaths(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' || r == '\t' })
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		path := normalizePath(part)
		if path != "" {
			out = append(out, path)
		}
	}
	return out
}
