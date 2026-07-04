package journalguard

import (
	"bufio"
	"context"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cedar2025/xboard-node/internal/nodeguard"
)

type Config struct {
	PanelURL  string
	Token     string
	NodeID    int
	Unit      string
	Transport string
	Since     string
	Throttle  time.Duration
}

type Agent struct {
	Config   Config
	Reporter nodeguard.Reporter
	mu       sync.Mutex
	last     map[string]time.Time
}

func New(c Config, r nodeguard.Reporter) *Agent {
	if c.Unit == "" {
		c.Unit = "xboard-node"
	}
	if c.Transport == "" {
		c.Transport = "tcp_http"
	}
	if c.Since == "" {
		c.Since = "now"
	}
	if c.Throttle <= 0 {
		c.Throttle = 60 * time.Second
	}
	return &Agent{Config: c, Reporter: r, last: map[string]time.Time{}}
}

func (a *Agent) Follow(ctx context.Context) error {
	args := []string{"-f", "-u", a.Config.Unit, "--no-pager", "-o", "short-iso"}
	if strings.TrimSpace(a.Config.Since) != "" {
		args = append(args, "--since", a.Config.Since)
	}
	cmd := exec.CommandContext(ctx, "journalctl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go copyJournalStderr(stderr)
	err = a.Process(ctx, stdout)
	if waitErr := cmd.Wait(); err == nil {
		err = waitErr
	}
	return err
}

func (a *Agent) Process(ctx context.Context, r io.Reader) error {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for s.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := s.Text()
		p, ok := ParseRejectLine(line, time.Now())
		if !ok || !a.allow(p) {
			continue
		}
		payload := p.Payload(a.Config.Token, a.Config.NodeID, a.Config.Transport)
		if err := a.Reporter.Report(payload); err != nil {
			log.Printf("journal guard report failed: %v", err)
		}
	}
	return s.Err()
}

func (a *Agent) allow(p ParsedReject) bool {
	key := p.ActualHost
	if key == "" {
		key = p.ActualSNI
	}
	if key == "" {
		key = p.ActualPath
	}
	if key == "" {
		key = p.RawReason
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	if last, ok := a.last[key]; ok && now.Sub(last) < a.Config.Throttle {
		return false
	}
	a.last[key] = now
	return true
}

func copyJournalStderr(r io.Reader) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line != "" {
			log.Printf("journalctl: %s", line)
		}
	}
}
