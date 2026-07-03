package nodeguard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type ReportPayload struct {
	Token        string `json:"token"`
	NodeID       int    `json:"node_id"`
	UserID       int    `json:"user_id,omitempty"`
	Email        string `json:"email,omitempty"`
	ClientIP     string `json:"client_ip,omitempty"`
	ExpectedHost string `json:"expected_host,omitempty"`
	ActualHost   string `json:"actual_host,omitempty"`
	ExpectedSNI  string `json:"expected_sni,omitempty"`
	ActualSNI    string `json:"actual_sni,omitempty"`
	ExpectedPath string `json:"expected_path,omitempty"`
	ActualPath   string `json:"actual_path,omitempty"`
	RawReason    string `json:"raw_reason,omitempty"`
	Action       string `json:"action,omitempty"`
	Confidence   string `json:"confidence,omitempty"`
	Transport    string `json:"transport,omitempty"`
}

func BuildReportPayload(c Config, ip string, a Actual, d Decision) ReportPayload {
	c = c.Normalize()
	return ReportPayload{
		Token:        c.ServerToken,
		NodeID:       c.NodeID,
		UserID:       c.UserID,
		Email:        c.Email,
		ClientIP:     ip,
		ExpectedHost: c.ExpectedHost,
		ActualHost:   a.Host,
		ExpectedSNI:  c.ExpectedSNI,
		ActualSNI:    a.SNI,
		ExpectedPath: normalizePath(c.ExpectedPath),
		ActualPath:   stripMarkerQuery(normalizePath(a.Path)),
		RawReason:    strings.Join(d.Reasons, "; "),
		Action:       d.Action,
		Confidence:   c.Confidence,
		Transport:    string(c.Transport),
	}
}

func (p ReportPayload) SafeJSON() ([]byte, error) { return json.Marshal(p) }

type Reporter interface{ Report(ReportPayload) error }
type HTTPReporter struct {
	PanelURL string
	Client   *http.Client
}

func (r HTTPReporter) Report(p ReportPayload) error {
	if r.PanelURL == "" {
		return nil
	}
	body, err := p.SafeJSON()
	if err != nil {
		return err
	}
	c := r.Client
	if c == nil {
		c = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(r.PanelURL, "/")+"/api/v2/server/config-guard/report", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
