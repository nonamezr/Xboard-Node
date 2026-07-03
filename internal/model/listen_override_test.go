package model

import (
	"testing"

	"github.com/cedar2025/xboard-node/internal/config"
	"github.com/cedar2025/xboard-node/internal/panel"
)

func TestNodeSpecFromPanelValidated_ListenOverrideIsLocalOnly(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vless",
		ListenIP:   "0.0.0.0",
		ServerPort: 443,
	}

	spec, err := NodeSpecFromPanelValidated(nc, config.KernelConfig{
		Type:               "singbox",
		ListenOverrideIP:   "127.0.0.1",
		ListenOverridePort: 8443,
	})
	if err != nil {
		t.Fatalf("NodeSpecFromPanelValidated: %v", err)
	}
	if spec.ServerPort != 443 || spec.ListenIP != "0.0.0.0" {
		t.Fatalf("panel/customer bind fields changed: listen=%q port=%d", spec.ListenIP, spec.ServerPort)
	}
	if spec.EffectiveListenIP() != "127.0.0.1" || spec.EffectiveListenPort() != 8443 {
		t.Fatalf("effective local bind = %q:%d, want 127.0.0.1:8443", spec.EffectiveListenIP(), spec.EffectiveListenPort())
	}
}

func TestNodeSpec_EffectiveListenDefaultsToPanelConfig(t *testing.T) {
	spec := &NodeSpec{ListenIP: "0.0.0.0", ServerPort: 443}
	if spec.EffectiveListenIP() != "0.0.0.0" || spec.EffectiveListenPort() != 443 {
		t.Fatalf("effective listen changed without override: %q:%d", spec.EffectiveListenIP(), spec.EffectiveListenPort())
	}
}
