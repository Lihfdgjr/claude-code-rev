package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"claudecode/internal/core"
)

type dnsLookupTool struct{}

type dnsLookupInput struct {
	Host       string `json:"host"`
	RecordType string `json:"record_type,omitempty"`
}

func NewDNSLookup() core.Tool { return &dnsLookupTool{} }

func (dnsLookupTool) Name() string { return "DNSLookup" }

func (dnsLookupTool) Description() string {
	return "Resolve DNS records. Supports A, AAAA, MX, TXT, CNAME, NS, and PTR (reverse) lookups."
}

func (dnsLookupTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "host": {"type": "string"},
    "record_type": {"type": "string", "enum": ["A", "AAAA", "MX", "TXT", "CNAME", "NS", "PTR"]}
  },
  "required": ["host"],
  "additionalProperties": false
}`)
}

func (dnsLookupTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in dnsLookupInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.Host == "" {
		return "", fmt.Errorf("host is required")
	}
	rt := strings.ToUpper(in.RecordType)
	if rt == "" {
		rt = "A"
	}

	var b strings.Builder
	switch rt {
	case "A", "AAAA":
		addrs, err := net.DefaultResolver.LookupHost(ctx, in.Host)
		if err != nil {
			return "", err
		}
		want4 := rt == "A"
		for _, a := range addrs {
			ip := net.ParseIP(a)
			if ip == nil {
				continue
			}
			is4 := ip.To4() != nil
			if want4 == is4 {
				fmt.Fprintf(&b, "%s\t%s\n", rt, a)
			}
		}
	case "MX":
		mxs, err := net.DefaultResolver.LookupMX(ctx, in.Host)
		if err != nil {
			return "", err
		}
		for _, m := range mxs {
			fmt.Fprintf(&b, "MX\t%d %s\n", m.Pref, m.Host)
		}
	case "TXT":
		txts, err := net.DefaultResolver.LookupTXT(ctx, in.Host)
		if err != nil {
			return "", err
		}
		for _, t := range txts {
			fmt.Fprintf(&b, "TXT\t%s\n", t)
		}
	case "CNAME":
		cn, err := net.DefaultResolver.LookupCNAME(ctx, in.Host)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "CNAME\t%s\n", cn)
	case "NS":
		nss, err := net.DefaultResolver.LookupNS(ctx, in.Host)
		if err != nil {
			return "", err
		}
		for _, n := range nss {
			fmt.Fprintf(&b, "NS\t%s\n", n.Host)
		}
	case "PTR":
		names, err := net.DefaultResolver.LookupAddr(ctx, in.Host)
		if err != nil {
			return "", err
		}
		for _, n := range names {
			fmt.Fprintf(&b, "PTR\t%s\n", n)
		}
	default:
		return "", fmt.Errorf("unsupported record_type: %s", rt)
	}
	out := b.String()
	if out == "" {
		return fmt.Sprintf("no %s records for %s\n", rt, in.Host), nil
	}
	return out, nil
}
