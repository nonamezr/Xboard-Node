package model

// EffectiveListenIP returns the local bind IP for generated kernel inbounds.
// It preserves the panel/customer ListenIP unless a local override is set.
func (n *NodeSpec) EffectiveListenIP() string {
	if n == nil {
		return ""
	}
	if n.LocalListenIP != "" {
		return n.LocalListenIP
	}
	return n.ListenIP
}

// EffectiveListenPort returns the local bind port for generated kernel inbounds.
// It preserves the panel/customer ServerPort unless a local override is set.
func (n *NodeSpec) EffectiveListenPort() int {
	if n == nil {
		return 0
	}
	if n.LocalListenPort > 0 {
		return n.LocalListenPort
	}
	return n.ServerPort
}
