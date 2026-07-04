package journalguard

import (
	"testing"
	"time"
)

func TestParseBadHostLine(t *testing.T) {
	now := time.Unix(1783134000, 0).UTC()
	line := "2026-07-04T03:07:25+00:00 vultr xboard-node[1]: +0000 2026-07-04 03:07:25 ERROR inbound/vless[vless-in]: bad host: www.example123.com"
	got, ok := ParseRejectLine(line, now)
	if !ok {
		t.Fatal("expected parse")
	}
	if got.ActualHost != "www.example123.com" {
		t.Fatalf("ActualHost=%q", got.ActualHost)
	}
	if got.ClientIP != "" {
		t.Fatalf("ClientIP=%q", got.ClientIP)
	}
	if got.RawReason != line {
		t.Fatal("raw reason should preserve original line")
	}
	if got.ReportedAt.Unix() != 1783134445 {
		t.Fatalf("ReportedAt=%v", got.ReportedAt)
	}
}

func TestParseBadHostLineWithClientIP(t *testing.T) {
	line := "2026-07-04T03:32:01+00:00 vultr xboard-node[95225]: tcp:127.0.0.1:443 accepted tcp:example.com:443 [vless-in >> proxy] email: user32@example.com process connection from 1.53.55.198:58123: bad host: www.example123.com"
	got, ok := ParseRejectLine(line, time.Unix(100, 0))
	if !ok {
		t.Fatal("expected parse")
	}
	if got.ActualHost != "www.example123.com" {
		t.Fatalf("ActualHost=%q", got.ActualHost)
	}
	if got.ClientIP != "1.53.55.198" {
		t.Fatalf("ClientIP=%q", got.ClientIP)
	}
}

func TestParseValidateHostLine(t *testing.T) {
	line := "transport/internet/websocket: failed to validate host, request:157.66.101.210:443, config:www.abc.com"
	got, ok := ParseRejectLine(line, time.Unix(100, 0))
	if !ok {
		t.Fatal("expected parse")
	}
	if got.ActualHost != "157.66.101.210" {
		t.Fatalf("ActualHost=%q", got.ActualHost)
	}
	if got.ReportedAt.Unix() != 100 {
		t.Fatalf("fallback ReportedAt=%v", got.ReportedAt)
	}
}

func TestParsePathAndSNI(t *testing.T) {
	line := "reject reason path=/bad sni=wrong.example.com"
	got, ok := ParseRejectLine(line, time.Unix(100, 0))
	if !ok {
		t.Fatal("expected parse")
	}
	if got.ActualPath != "/bad" || got.ActualSNI != "wrong.example.com" {
		t.Fatalf("path=%q sni=%q", got.ActualPath, got.ActualSNI)
	}
}

func TestParseIgnoresNoise(t *testing.T) {
	if _, ok := ParseRejectLine("report pushed: 1 users, 1 online", time.Now()); ok {
		t.Fatal("noise parsed as reject")
	}
}

func TestPayloadIsMonitorOnly(t *testing.T) {
	p, ok := ParseRejectLine("bad host: bad.example.com", time.Unix(123, 0))
	if !ok {
		t.Fatal("expected parse")
	}
	payload := p.Payload("secret-token", 70, "tcp_http")
	if payload.Action != "monitor" || payload.NodeID != 70 || payload.ActualHost != "bad.example.com" || payload.ReportedAt != 123 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.ClientIP != "" || payload.UserID != 0 {
		t.Fatalf("journal payload should not invent missing user/client ip: %+v", payload)
	}
}

func TestPayloadIncludesParsedClientIP(t *testing.T) {
	p, ok := ParseRejectLine("process connection from 1.53.55.198:58123: bad host: bad.example.com", time.Unix(123, 0))
	if !ok {
		t.Fatal("expected parse")
	}
	payload := p.Payload("secret-token", 70, "tcp_http")
	if payload.ClientIP != "1.53.55.198" {
		t.Fatalf("ClientIP=%q", payload.ClientIP)
	}
}
