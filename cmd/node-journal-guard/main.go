package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cedar2025/xboard-node/internal/journalguard"
	"github.com/cedar2025/xboard-node/internal/nodeguard"
)

const tokenFileEnv = "NODEGUARD_SERVER_TOKEN_FILE"

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

func loadToken(flagTokenFile string) (string, error) {
	if flagTokenFile != "" {
		return readTokenFile(flagTokenFile)
	}
	if envTokenFile := strings.TrimSpace(os.Getenv(tokenFileEnv)); envTokenFile != "" {
		return readTokenFile(envTokenFile)
	}
	return "", fmt.Errorf("missing token file: pass -token-file or set %s", tokenFileEnv)
}

func main() {
	var panelURL, tokenFile, unit, transport, since string
	var nodeID int
	var throttle time.Duration
	flag.StringVar(&panelURL, "panel", "", "panel base URL")
	flag.StringVar(&tokenFile, "token-file", "", "path to server token file; overrides NODEGUARD_SERVER_TOKEN_FILE")
	flag.IntVar(&nodeID, "node-id", 0, "node id")
	flag.StringVar(&unit, "unit", "xboard-node", "systemd journal unit to follow")
	flag.StringVar(&transport, "transport", "tcp_http", "transport label for reports")
	flag.StringVar(&since, "since", "now", "journalctl --since value")
	flag.DurationVar(&throttle, "dedup", 60*time.Second, "dedup window per host/sni/path")
	flag.Parse()

	if panelURL == "" {
		log.Fatal("missing -panel")
	}
	if nodeID <= 0 {
		log.Fatal("missing -node-id")
	}
	token, err := loadToken(tokenFile)
	if err != nil {
		log.Fatalf("load server token: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	a := journalguard.New(journalguard.Config{PanelURL: panelURL, Token: token, NodeID: nodeID, Unit: unit, Transport: transport, Since: since, Throttle: throttle}, nodeguard.HTTPReporter{PanelURL: panelURL})
	log.Printf("node-journal-guard unit=%s node_id=%d transport=%s panel=%s dedup=%s", unit, nodeID, transport, strings.TrimRight(panelURL, "/"), throttle)
	if err := a.Follow(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
