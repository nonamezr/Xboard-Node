package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cedar2025/xboard-node/internal/nodeguard"
)

const tokenFileEnv = "NODEGUARD_SERVER_TOKEN_FILE"

func loadToken(flagToken, flagTokenFile string) (string, error) {
	if flagTokenFile != "" {
		return readTokenFile(flagTokenFile)
	}
	if envTokenFile := strings.TrimSpace(os.Getenv(tokenFileEnv)); envTokenFile != "" {
		return readTokenFile(envTokenFile)
	}
	if flagToken != "" {
		log.Printf("warning: -token exposes secrets in process args; prefer -token-file or %s", tokenFileEnv)
	}
	return strings.TrimSpace(flagToken), nil
}

func readTokenFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("token file path is empty")
	}
	st, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat token file: %w", err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("token file is a directory")
	}
	if st.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("token file permissions must not allow group/other access; expected 0600 or stricter")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read token file: %w", err)
	}
	token := strings.TrimSpace(string(b))
	if token == "" {
		return "", fmt.Errorf("token file is empty")
	}
	return token, nil
}

func main() {
	var c nodeguard.Config
	var flagToken string
	var flagTokenFile string
	flag.StringVar((*string)(&c.Mode), "mode", string(nodeguard.ModeMonitor), "off|monitor_only|reject_only")
	flag.StringVar((*string)(&c.Transport), "transport", string(nodeguard.TransportTCPHTTP), "tcp_http|ws_tls")
	flag.StringVar(&c.Listen, "listen", "127.0.0.1:9443", "listen address")
	flag.StringVar(&c.Backend, "backend", "127.0.0.1:9444", "backend address or URL")
	flag.StringVar(&c.ExpectedHost, "host", "www.abc.com", "expected host")
	flag.StringVar(&c.ExpectedSNI, "sni", "", "expected sni")
	flag.StringVar(&c.ExpectedPath, "path", "/vpn", "expected path")
	legacyPaths := flag.String("legacy-paths", "/,/vpn", "comma/newline separated legacy paths allowed for path comparison")
	flag.StringVar(&c.PanelURL, "panel", "", "panel base URL")
	flag.StringVar(&flagToken, "token", "", "server token (deprecated: prefer -token-file or NODEGUARD_SERVER_TOKEN_FILE)")
	flag.StringVar(&flagTokenFile, "token-file", "", "path to server token file; overrides NODEGUARD_SERVER_TOKEN_FILE and -token")
	flag.IntVar(&c.NodeID, "node-id", 0, "node id")
	flag.IntVar(&c.UserID, "user-id", 0, "optional user id")
	flag.StringVar(&c.Email, "email", "", "optional email")
	flag.StringVar(&c.Protocol, "protocol", "", "protocol label")
	flag.StringVar(&c.Confidence, "confidence", "", "confidence label")
	flag.StringVar(&c.TLSCertFile, "cert", "", "TLS cert for ws_tls")
	flag.StringVar(&c.TLSKeyFile, "key", "", "TLS key for ws_tls")
	flag.Parse()
	serverToken, err := loadToken(flagToken, flagTokenFile)
	if err != nil {
		log.Fatalf("load server token: %v", err)
	}
	c.ServerToken = serverToken
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
