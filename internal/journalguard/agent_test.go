package journalguard

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cedar2025/xboard-node/internal/nodeguard"
)

type captureReporter struct{ payloads []nodeguard.ReportPayload }

func (c *captureReporter) Report(p nodeguard.ReportPayload) error {
	c.payloads = append(c.payloads, p)
	return nil
}

func TestAgentDedupsSameHost(t *testing.T) {
	r := &captureReporter{}
	a := New(Config{Token: "token", NodeID: 70, Transport: "tcp_http", Throttle: time.Minute}, r)
	input := strings.NewReader("bad host: bad.example.com\nbad host: bad.example.com\nbad host: other.example.com\n")
	if err := a.Process(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if len(r.payloads) != 2 {
		t.Fatalf("payload count=%d", len(r.payloads))
	}
	if r.payloads[0].ActualHost != "bad.example.com" || r.payloads[1].ActualHost != "other.example.com" {
		t.Fatalf("payloads=%+v", r.payloads)
	}
}
