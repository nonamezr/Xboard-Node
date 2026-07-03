package nodeguard

import (
	"bufio"
	"bytes"
	"errors"
	"net/textproto"
	"strings"
)

type Preface struct {
	Host string
	Path string
	Raw  []byte
}

func ParseHTTPPreface(raw []byte) (Preface, error) {
	idx := bytes.Index(raw, []byte("\r\n\r\n"))
	if idx < 0 {
		return Preface{Raw: raw}, errors.New("incomplete http preface")
	}
	head := raw[:idx+4]
	br := bufio.NewReader(bytes.NewReader(head))
	line, err := br.ReadString('\n')
	if err != nil {
		return Preface{Raw: raw}, err
	}
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) < 2 {
		return Preface{Raw: raw}, errors.New("malformed request line")
	}
	tp := textproto.NewReader(br)
	hdr, err := tp.ReadMIMEHeader()
	if err != nil {
		return Preface{Raw: raw}, err
	}
	host := hdr.Get("Host")
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	return Preface{Host: host, Path: parts[1], Raw: raw}, nil
}

func RedactSecretMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "token") || strings.Contains(lk, "password") || strings.Contains(lk, "secret") || strings.Contains(lk, "uuid") || strings.Contains(lk, "private") || strings.Contains(lk, "key") {
			out[k] = "[REDACTED]"
			continue
		}
		out[k] = v
	}
	return out
}
