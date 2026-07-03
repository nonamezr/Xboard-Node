package nodeguard

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Guard struct {
	Config   Config
	Reporter Reporter
	sniMu    sync.Mutex
	sni      map[string]string
}

func New(c Config, r Reporter) *Guard {
	c = c.Normalize()
	return &Guard{Config: c, Reporter: r, sni: map[string]string{}}
}
func clientIP(remote string) string {
	h, _, err := net.SplitHostPort(remote)
	if err == nil {
		return h
	}
	return remote
}

func (g *Guard) ServeTCPHTTP(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go g.handleTCP(c)
	}
}
func (g *Guard) handleTCP(c net.Conn) {
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 4096)
	for !strings.Contains(string(buf), "\r\n\r\n") && len(buf) < 8192 {
		n, err := c.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			return
		}
	}
	_ = c.SetReadDeadline(time.Time{})
	p, err := ParseHTTPPreface(buf)
	if err != nil {
		return
	}
	actual := Actual{Host: p.Host, Path: p.Path}
	d := Evaluate(g.Config, actual)
	if !d.Match && g.Config.Mode != ModeOff {
		_ = g.Reporter.Report(BuildReportPayload(g.Config, clientIP(c.RemoteAddr().String()), actual, d))
	}
	if !d.Match && g.Config.Mode == ModeReject {
		return
	}
	b, err := net.DialTimeout("tcp", g.Config.Backend, 5*time.Second)
	if err != nil {
		return
	}
	defer b.Close()
	_, _ = b.Write(buf)
	go io.Copy(b, c)
	io.Copy(c, b)
}

func (g *Guard) ListenAndServeTCPHTTP() error {
	ln, err := net.Listen("tcp", g.Config.Listen)
	if err != nil {
		return err
	}
	return g.ServeTCPHTTP(ln)
}

func (g *Guard) ListenAndServeWSTLS() error {
	cert, err := tls.LoadX509KeyPair(g.Config.TLSCertFile, g.Config.TLSKeyFile)
	if err != nil {
		return err
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	cfg.GetConfigForClient = func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
		if chi.Conn != nil {
			g.sniMu.Lock()
			g.sni[chi.Conn.RemoteAddr().String()] = chi.ServerName
			g.sniMu.Unlock()
		}
		return nil, nil
	}
	srv := &http.Server{Addr: g.Config.Listen, Handler: g, TLSConfig: cfg, ReadHeaderTimeout: 5 * time.Second}
	return srv.ListenAndServeTLS("", "")
}
func (g *Guard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remote := r.RemoteAddr
	g.sniMu.Lock()
	sni := g.sni[remote]
	delete(g.sni, remote)
	g.sniMu.Unlock()
	host := strings.Split(r.Host, ":")[0]
	actual := Actual{Host: host, SNI: sni, Path: r.URL.Path}
	d := Evaluate(g.Config, actual)
	if !d.Match && g.Config.Mode != ModeOff {
		_ = g.Reporter.Report(BuildReportPayload(g.Config, clientIP(remote), actual, d))
	}
	if !d.Match && g.Config.Mode == ModeReject {
		http.Error(w, "node config guard reject", http.StatusForbidden)
		return
	}
	u, err := url.Parse(g.Config.Backend)
	if err != nil {
		http.Error(w, "bad backend", 500)
		return
	}
	p := httputil.NewSingleHostReverseProxy(u)
	od := p.Director
	p.Director = func(req *http.Request) { od(req); req.Host = host; req.URL.Path = r.URL.Path }
	p.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, ServerName: g.Config.ExpectedSNI}}
	p.ErrorLog = log.Default()
	p.ServeHTTP(w, r)
}
