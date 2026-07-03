package main

import (
	"flag"
	"github.com/cedar2025/xboard-node/internal/nodeguard"
	"log"
)

func main() {
	var c nodeguard.Config
	flag.StringVar((*string)(&c.Mode), "mode", string(nodeguard.ModeMonitor), "off|monitor_only|reject_only")
	flag.StringVar((*string)(&c.Transport), "transport", string(nodeguard.TransportTCPHTTP), "tcp_http|ws_tls")
	flag.StringVar(&c.Listen, "listen", "127.0.0.1:9443", "listen address")
	flag.StringVar(&c.Backend, "backend", "127.0.0.1:9444", "backend address or URL")
	flag.StringVar(&c.ExpectedHost, "host", "www.abc.com", "expected host")
	flag.StringVar(&c.ExpectedSNI, "sni", "", "expected sni")
	flag.StringVar(&c.ExpectedPath, "path", "/vpn", "expected path")
	legacyPaths := flag.String("legacy-paths", "/,/vpn", "comma/newline separated legacy paths allowed for path comparison")
	flag.StringVar(&c.PanelURL, "panel", "", "panel base URL")
	flag.StringVar(&c.ServerToken, "token", "", "server token")
	flag.IntVar(&c.NodeID, "node-id", 0, "node id")
	flag.IntVar(&c.UserID, "user-id", 0, "optional user id")
	flag.StringVar(&c.Email, "email", "", "optional email")
	flag.StringVar(&c.Protocol, "protocol", "", "protocol label")
	flag.StringVar(&c.Confidence, "confidence", "", "confidence label")
	flag.StringVar(&c.TLSCertFile, "cert", "", "TLS cert for ws_tls")
	flag.StringVar(&c.TLSKeyFile, "key", "", "TLS key for ws_tls")
	flag.Parse()
	c.LegacyPaths = nodeguard.SplitLegacyPaths(*legacyPaths)
	c = c.Normalize()
	if !c.Mode.Valid() {
		log.Fatalf("invalid mode %s", c.Mode)
	}
	g := nodeguard.New(c, nodeguard.HTTPReporter{PanelURL: c.PanelURL})
	log.Printf("node-config-guard mode=%s transport=%s listen=%s backend=%s", c.Mode, c.Transport, c.Listen, c.Backend)
	if c.Transport == nodeguard.TransportWSTLS {
		log.Fatal(g.ListenAndServeWSTLS())
	}
	log.Fatal(g.ListenAndServeTCPHTTP())
}
