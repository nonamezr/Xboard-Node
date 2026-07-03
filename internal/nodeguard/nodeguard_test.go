package nodeguard

import (
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

type recReporter struct{ payloads []ReportPayload }

func (r *recReporter) Report(p ReportPayload) error { r.payloads = append(r.payloads, p); return nil }

func TestParseHTTPPreface(t *testing.T) {
	raw := []byte("GET /vpn HTTP/1.1\r\nHost: www.abc.com:443\r\nX-Test: ok\r\n\r\nTAIL")
	p, err := ParseHTTPPreface(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Host != "www.abc.com" || p.Path != "/vpn" {
		t.Fatalf("host/path=%q/%q", p.Host, p.Path)
	}
	if !bytes.Equal(p.Raw, raw) {
		t.Fatal("raw bytes changed")
	}
}
func TestParseMalformedHeader(t *testing.T) {
	if _, err := ParseHTTPPreface([]byte("GET /vpn HTTP/1.1\r\nHost: x")); err == nil {
		t.Fatal("expected error")
	}
}
func TestEvaluateModes(t *testing.T) {
	c := Config{ExpectedHost: "www.abc.com", ExpectedPath: "/vpn", Mode: ModeOff}
	if d := Evaluate(c, Actual{Host: "bad", Path: "/vpn"}); d.Action != "allow" || d.Match {
		t.Fatalf("off decision=%+v", d)
	}
	c.Mode = ModeMonitor
	if d := Evaluate(c, Actual{Host: "bad", Path: "/vpn"}); d.Action != "monitor" || d.Match {
		t.Fatalf("monitor decision=%+v", d)
	}
	c.Mode = ModeReject
	if d := Evaluate(c, Actual{Host: "bad", Path: "/vpn"}); d.Action != "reject" || d.Match {
		t.Fatalf("reject decision=%+v", d)
	}
}
func TestSecretMaskingAndReportPayload(t *testing.T) {
	masked := RedactSecretMap(map[string]any{"uuid": "raw-uuid", "token": "raw-token", "password": "pw", "host": "ok"})
	if masked["uuid"] == "raw-uuid" || masked["token"] == "raw-token" || masked["password"] == "pw" {
		t.Fatalf("not masked: %#v", masked)
	}
	p := BuildReportPayload(Config{ServerToken: "server-token", NodeID: 7, UserID: 9, ExpectedHost: "good", ExpectedPath: "/vpn", Transport: TransportTCPHTTP, Mode: ModeMonitor}, "127.0.0.1", Actual{Host: "bad", Path: "/wrong"}, Decision{Reasons: []string{"host mismatch"}, Action: "monitor"})
	b, err := p.SafeJSON()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "raw-uuid") || strings.Contains(s, "password") {
		t.Fatalf("payload leaked secret: %s", s)
	}
	if p.ExpectedHost != "good" || p.ActualHost != "bad" || p.ExpectedPath != "/vpn" || p.ActualPath != "/wrong" || p.Action != "monitor" || p.Confidence != "high" || p.Transport != "tcp_http" {
		t.Fatalf("unexpected report payload: %+v", p)
	}
}
func TestTCPGuardForwardUnchangedOnMatch(t *testing.T) {
	backendLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backendLn.Close()
	got := make(chan []byte, 1)
	go func() {
		c, _ := backendLn.Accept()
		defer c.Close()
		buf := make([]byte, 128)
		n, _ := c.Read(buf)
		got <- buf[:n]
		_, _ = c.Write([]byte("OK"))
	}()
	guardLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer guardLn.Close()
	rr := &recReporter{}
	g := New(Config{Mode: ModeReject, Listen: guardLn.Addr().String(), Backend: backendLn.Addr().String(), ExpectedHost: "www.abc.com", ExpectedPath: "/vpn"}, rr)
	go g.ServeTCPHTTP(guardLn)
	time.Sleep(50 * time.Millisecond)
	raw := []byte("GET /vpn HTTP/1.1\r\nHost: www.abc.com\r\n\r\nVMESSBYTES")
	c, err := net.Dial("tcp", guardLn.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, _ = c.Write(raw)
	resp := make([]byte, 2)
	if _, err := io.ReadFull(c, resp); err != nil {
		t.Fatal(err)
	}
	if string(resp) != "OK" {
		t.Fatalf("resp=%q", resp)
	}
	if !bytes.Equal(<-got, raw) {
		t.Fatal("forwarded bytes changed")
	}
	if len(rr.payloads) != 0 {
		t.Fatal("unexpected report on match")
	}
}
func TestTCPGuardRejectMismatchAndMonitor(t *testing.T) {
	for _, mode := range []Mode{ModeReject, ModeMonitor, ModeOff} {
		backendLn, _ := net.Listen("tcp", "127.0.0.1:0")
		defer backendLn.Close()
		hit := make(chan bool, 1)
		go func() {
			c, err := backendLn.Accept()
			if err == nil {
				hit <- true
				c.Close()
			}
		}()
		guardLn, _ := net.Listen("tcp", "127.0.0.1:0")
		defer guardLn.Close()
		rr := &recReporter{}
		g := New(Config{Mode: mode, Backend: backendLn.Addr().String(), ExpectedHost: "www.abc.com", ExpectedPath: "/vpn"}, rr)
		go g.ServeTCPHTTP(guardLn)
		time.Sleep(20 * time.Millisecond)
		c, _ := net.Dial("tcp", guardLn.Addr().String())
		_, _ = c.Write([]byte("GET /vpn HTTP/1.1\r\nHost: bad.example\r\n\r\n"))
		_ = c.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
		_, _ = c.Read(make([]byte, 1))
		c.Close()
		if mode == ModeReject && len(rr.payloads) != 1 {
			t.Fatalf("reject reports=%d", len(rr.payloads))
		}
		if mode == ModeMonitor && len(rr.payloads) != 1 {
			t.Fatalf("monitor reports=%d", len(rr.payloads))
		}
		if mode == ModeOff && len(rr.payloads) != 0 {
			t.Fatalf("off reports=%d", len(rr.payloads))
		}
		select {
		case <-hit:
			if mode == ModeReject {
				t.Fatal("reject forwarded to backend")
			}
		default:
			if mode != ModeReject {
				t.Fatal("non-reject did not forward")
			}
		}
	}
}
